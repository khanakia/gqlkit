/** Options for configuring the GraphQL client */
export interface ClientOptions {
    headers?: Record<string, string>;
    authToken?: string;
    fetch?: typeof fetch;
}
/** Represents a single GraphQL error */
export interface GraphQLError {
    message: string;
    locations?: {
        line: number;
        column: number;
    }[];
    path?: (string | number)[];
    extensions?: Record<string, unknown>;
}
/** Error class for GraphQL error responses */
export declare class GraphQLErrors extends Error {
    errors: GraphQLError[];
    constructor(errors: GraphQLError[]);
}
/** GraphQL HTTP client */
export declare class GraphQLClient {
    private endpoint;
    private options;
    constructor(endpoint: string, options?: ClientOptions);
    /** Execute a GraphQL operation */
    execute<T>(query: string, variables?: Record<string, unknown>): Promise<T>;
    /** Execute a raw GraphQL query */
    rawQuery(query: string, variables?: Record<string, unknown>): Promise<unknown>;
}
