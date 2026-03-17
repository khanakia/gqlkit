/**
 * @module graphqlclient
 *
 * Provides the low-level HTTP transport for executing GraphQL operations.
 * Handles JSON serialization, header management, auth tokens, and error extraction.
 */

/**
 * Configuration options for {@link GraphQLClient}.
 *
 * @property headers   - Additional HTTP headers merged into every request.
 * @property authToken - Bearer token appended as an `Authorization` header.
 * @property fetch     - Custom fetch implementation (useful for SSR or testing).
 */
export interface ClientOptions {
  headers?: Record<string, string>;
  authToken?: string;
  fetch?: typeof fetch;
}

/**
 * Represents a single error entry from a GraphQL response's `errors` array.
 * Follows the GraphQL spec error format.
 *
 * @property message    - Human-readable error description.
 * @property locations  - Source locations in the query that caused the error.
 * @property path       - The response field path where the error occurred.
 * @property extensions - Vendor-specific metadata (e.g., error codes).
 */
export interface GraphQLError {
  message: string;
  locations?: { line: number; column: number }[];
  path?: (string | number)[];
  extensions?: Record<string, unknown>;
}

/**
 * Custom error class thrown when the GraphQL server returns one or more errors.
 * The `message` property is a semicolon-joined summary of all error messages.
 * The original error objects are available via the `errors` property.
 */
export class GraphQLErrors extends Error {
  /** The raw GraphQL error objects from the server response. */
  public errors: GraphQLError[];

  /**
   * @param errors - Array of GraphQL error objects from the response.
   */
  constructor(errors: GraphQLError[]) {
    // Combine all error messages into a single string for the Error superclass
    const message = errors.map((e) => e.message).join("; ");
    super(message);
    this.name = "GraphQLErrors";
    this.errors = errors;
  }
}

/**
 * HTTP client for executing GraphQL queries and mutations against a single endpoint.
 *
 * Responsibilities:
 * - Sends POST requests with JSON-encoded `{ query, variables }` bodies.
 * - Merges custom headers and optional Bearer auth token into each request.
 * - Parses the JSON response, throwing {@link GraphQLErrors} when the server reports errors.
 * - Supports a custom `fetch` function for environments without a global `fetch`.
 *
 * @example
 * ```ts
 * const client = new GraphQLClient("https://api.example.com/graphql", {
 *   authToken: "my-token",
 * });
 * const data = await client.execute<{ user: User }>(query, variables);
 * ```
 */
export class GraphQLClient {
  /** The GraphQL endpoint URL (e.g., "https://api.example.com/graphql"). */
  private endpoint: string;

  /** Client configuration including headers, auth, and optional custom fetch. */
  private options: ClientOptions;

  /**
   * @param endpoint - The URL of the GraphQL server.
   * @param options  - Optional client configuration.
   */
  constructor(endpoint: string, options?: ClientOptions) {
    this.endpoint = endpoint;
    this.options = options || {};
  }

  /**
   * Execute a GraphQL operation and return the typed `data` payload.
   *
   * Flow:
   * 1. Resolve the fetch function (custom or global).
   * 2. Build headers: Content-Type + user headers + optional Bearer token.
   * 3. POST the JSON body `{ query, variables }` to the endpoint.
   * 4. Parse the JSON response and check for errors.
   * 5. Return `response.data` or throw on errors / missing data.
   *
   * @typeParam T - The expected shape of `response.data`.
   * @param query     - The GraphQL query or mutation string.
   * @param variables - Optional variables map for the operation.
   * @returns The `data` field from the GraphQL response, typed as `T`.
   * @throws {GraphQLErrors} If the response contains a non-empty `errors` array.
   * @throws {Error} If the response has no `data` field at all.
   */
  async execute<T>(
    query: string,
    variables?: Record<string, unknown>
  ): Promise<T> {
    // Use the custom fetch if provided, otherwise fall back to the global fetch
    const fetchFn = this.options.fetch || globalThis.fetch;

    // Merge default Content-Type with any user-supplied headers
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...this.options.headers,
    };

    // Attach Bearer token if configured
    if (this.options.authToken) {
      headers["Authorization"] = `Bearer ${this.options.authToken}`;
    }

    // Send the GraphQL operation as a JSON POST request
    const response = await fetchFn(this.endpoint, {
      method: "POST",
      headers,
      body: JSON.stringify({ query, variables }),
    });

    // Parse the JSON response conforming to the GraphQL-over-HTTP spec
    const json = (await response.json()) as {
      data?: T;
      errors?: GraphQLError[];
    };

    // If the server returned errors, throw them as a structured exception
    if (json.errors && json.errors.length > 0) {
      throw new GraphQLErrors(json.errors);
    }

    // Guard against responses that have neither data nor errors
    if (!json.data) {
      throw new Error("No data returned from GraphQL query");
    }

    return json.data;
  }

  /**
   * Execute a raw GraphQL query without type inference on the result.
   * Convenience wrapper around {@link execute} for ad-hoc / untyped queries.
   *
   * @param query     - The GraphQL query or mutation string.
   * @param variables - Optional variables map for the operation.
   * @returns The untyped `data` payload from the GraphQL response.
   */
  async rawQuery(
    query: string,
    variables?: Record<string, unknown>
  ): Promise<unknown> {
    return this.execute(query, variables);
  }
}
