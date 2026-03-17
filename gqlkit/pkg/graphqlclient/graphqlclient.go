// Package graphqlclient provides a lightweight HTTP client for executing GraphQL
// queries and mutations against a remote endpoint. It implements the
// builder.GraphQLClient interface so it can be used directly with the generated
// SDK builders. Configuration is done via functional options (WithHTTPClient,
// WithHeader, WithAuthToken, etc.).
package graphqlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client sends GraphQL requests over HTTP to a single endpoint. It holds
// persistent headers (e.g. auth tokens) and an optional custom http.Client.
type Client struct {
	endpoint   string            // full URL of the GraphQL endpoint
	httpClient *http.Client      // underlying HTTP transport; defaults to http.DefaultClient
	headers    map[string]string // headers attached to every outgoing request
}

// ClientOption is a functional option for configuring a Client at creation time.
type ClientOption func(*Client)

// NewClient creates a new GraphQL client pointed at the given endpoint.
// Any number of ClientOption values can be passed to customise the client.
func NewClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		endpoint:   endpoint,
		httpClient: http.DefaultClient,
		headers:    make(map[string]string),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithHTTPClient sets the HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithHeader adds a header to all requests
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithHeaders adds multiple headers to all requests
func WithHeaders(headers map[string]string) ClientOption {
	return func(c *Client) {
		for key, value := range headers {
			c.headers[key] = value
		}
	}
}

// WithAuthToken adds an authorization bearer token
func WithAuthToken(token string) ClientOption {
	return func(c *Client) {
		c.headers["Authorization"] = "Bearer " + token
	}
}

// graphQLRequest is the JSON body sent to the GraphQL endpoint.
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLError represents a single error entry returned in a GraphQL response.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLErrorLocation `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLErrorLocation represents the location of a GraphQL error
type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Error implements the error interface
func (e GraphQLError) Error() string {
	return e.Message
}

// graphQLResponse is the top-level JSON structure returned by a GraphQL endpoint.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLErrors is a slice of GraphQLError that implements the error interface,
// joining all messages when there are multiple errors.
type GraphQLErrors []GraphQLError

// Error implements the error interface
func (e GraphQLErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Message
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Message)
	}
	return fmt.Sprintf("multiple errors: %v", msgs)
}

// Execute sends a GraphQL request (query or mutation) to the configured endpoint.
// The response data is JSON-unmarshalled into result. If the response contains
// GraphQL-level errors, they are returned as a GraphQLErrors value.
func (c *Client) Execute(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return GraphQLErrors(gqlResp.Errors)
	}

	if result != nil && gqlResp.Data != nil {
		if err := json.Unmarshal(gqlResp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return nil
}

// RawQuery executes a GraphQL query and returns the raw JSON "data" payload
// without unmarshalling it into a concrete type. Useful for dynamic queries.
func (c *Client) RawQuery(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	var result struct {
		Data json.RawMessage `json:"data"`
	}

	if err := c.Execute(ctx, query, variables, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
