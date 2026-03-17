package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// IntrospectionQuery is the standard GraphQL introspection query that retrieves
// the full schema definition. It uses fragments to recursively resolve type
// references up to 7 levels deep (sufficient for most real-world schemas with
// nested NON_NULL/LIST wrappers like [String!]!).
const IntrospectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type {
    ...TypeRef
  }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
            }
          }
        }
      }
    }
  }
}
`

// FetchOptions holds optional configuration for the schema fetch request,
// primarily custom HTTP headers (e.g., Authorization, Referer, Origin).
type FetchOptions struct {
	Headers map[string]string
}

// FetchSchema sends the introspection query to the given GraphQL endpoint URL
// via HTTP POST and returns the parsed IntrospectionSchema. It applies any
// custom headers from opts. Returns an error if the HTTP request fails, the
// response status is not 200, JSON parsing fails, or the GraphQL response
// contains errors.
func FetchSchema(url string, opts *FetchOptions) (*IntrospectionSchema, error) {
	// Build the JSON request body containing the introspection query.
	requestBody := map[string]any{
		"query": IntrospectionQuery,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP POST request to the GraphQL endpoint.
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Apply any user-supplied headers (auth tokens, referer, etc.).
	if opts != nil {
		for key, value := range opts.Headers {
			req.Header.Set(key, value)
		}
	}

	// Execute the HTTP request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Reject non-200 responses with the response body for debugging.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response into the IntrospectionResponse structure.
	var introspectionResp IntrospectionResponse
	if err := json.Unmarshal(body, &introspectionResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// If the GraphQL response contains errors, aggregate and return them.
	if len(introspectionResp.Errors) > 0 {
		messages := make([]string, len(introspectionResp.Errors))
		for i, e := range introspectionResp.Errors {
			messages[i] = e.Message
		}
		return nil, fmt.Errorf("GraphQL errors: %s", strings.Join(messages, "; "))
	}

	return &introspectionResp.Data.Schema, nil
}
