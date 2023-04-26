package gqlengineadapter

import (
	"encoding/json"
	"errors"
	"net/http"

	graphqlDataSource "github.com/TykTechnologies/graphql-go-tools/pkg/engine/datasource/graphql_datasource"
	restDataSource "github.com/TykTechnologies/graphql-go-tools/pkg/engine/datasource/rest_datasource"
	"github.com/TykTechnologies/graphql-go-tools/pkg/graphql"

	"github.com/TykTechnologies/tyk/apidef"
)

var (
	ErrGraphQLConfigIsMissingOperation = errors.New("graphql data source config is missing an operation")
)

func parseSchema(schemaAsString string) (parsedSchema *graphql.Schema, err error) {
	parsedSchema, err = graphql.NewSchemaFromString(schemaAsString)
	if err != nil {
		return nil, err
	}

	normalizationResult, err := parsedSchema.Normalize()
	if err != nil {
		return nil, err
	}

	if !normalizationResult.Successful && normalizationResult.Errors != nil {
		return nil, normalizationResult.Errors
	}

	return parsedSchema, nil
}

func graphqlDataSourceWebSocketProtocol(subscriptionType apidef.SubscriptionType) string {
	wsProtocol := graphqlDataSource.ProtocolGraphQLWS
	if subscriptionType == apidef.GQLSubscriptionTransportWS {
		wsProtocol = graphqlDataSource.ProtocolGraphQLTWS
	}
	return wsProtocol
}

func graphqlSubscriptionType(subscriptionType apidef.SubscriptionType) graphql.SubscriptionType {
	switch subscriptionType {
	case apidef.GQLSubscriptionWS:
		return graphql.SubscriptionTypeGraphQLWS
	case apidef.GQLSubscriptionTransportWS:
		return graphql.SubscriptionTypeGraphQLTransportWS
	case apidef.GQLSubscriptionSSE:
		return graphql.SubscriptionTypeSSE
	default:
		return graphql.SubscriptionTypeUnknown
	}
}

func convertApiDefinitionHeadersToHttpHeaders(apiDefHeaders map[string]string) http.Header {
	if len(apiDefHeaders) == 0 {
		return nil
	}

	engineV2Headers := make(http.Header)
	for apiDefHeaderKey, apiDefHeaderValue := range apiDefHeaders {
		engineV2Headers.Add(apiDefHeaderKey, apiDefHeaderValue)
	}

	return engineV2Headers
}

func removeDuplicateApiDefinitionHeaders(headers ...map[string]string) map[string]string {
	hdr := make(map[string]string)
	// headers priority depends on the order of arguments
	for _, header := range headers {
		for k, v := range header {
			keyCanonical := http.CanonicalHeaderKey(k)
			if _, ok := hdr[keyCanonical]; ok {
				// skip because header is present
				continue
			}
			hdr[keyCanonical] = v
		}
	}
	return hdr
}

func generateRestDataSourceFromGraphql(config apidef.GraphQLEngineDataSourceConfigGraphQL) (json.RawMessage, error) {
	if !config.HasOperation {
		return nil, ErrGraphQLConfigIsMissingOperation
	}
	req := graphql.Request{
		Query:     config.Operation,
		Variables: config.Variables,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	customMessage := restDataSource.ConfigJSON(restDataSource.Configuration{
		Fetch: restDataSource.FetchConfiguration{
			URL:    config.URL,
			Method: config.Method,
			Body:   string(body),
			Header: convertApiDefinitionHeadersToHttpHeaders(config.Headers),
		},
	})
	return customMessage, nil
}
