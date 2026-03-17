"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.GraphQLClient = exports.GraphQLErrors = void 0;
/** Error class for GraphQL error responses */
class GraphQLErrors extends Error {
    constructor(errors) {
        const message = errors.map((e) => e.message).join("; ");
        super(message);
        this.name = "GraphQLErrors";
        this.errors = errors;
    }
}
exports.GraphQLErrors = GraphQLErrors;
/** GraphQL HTTP client */
class GraphQLClient {
    constructor(endpoint, options) {
        this.endpoint = endpoint;
        this.options = options || {};
    }
    /** Execute a GraphQL operation */
    async execute(query, variables) {
        const fetchFn = this.options.fetch || globalThis.fetch;
        const headers = {
            "Content-Type": "application/json",
            ...this.options.headers,
        };
        if (this.options.authToken) {
            headers["Authorization"] = `Bearer ${this.options.authToken}`;
        }
        const response = await fetchFn(this.endpoint, {
            method: "POST",
            headers,
            body: JSON.stringify({ query, variables }),
        });
        const json = (await response.json());
        if (json.errors && json.errors.length > 0) {
            throw new GraphQLErrors(json.errors);
        }
        if (!json.data) {
            throw new Error("No data returned from GraphQL query");
        }
        return json.data;
    }
    /** Execute a raw GraphQL query */
    async rawQuery(query, variables) {
        return this.execute(query, variables);
    }
}
exports.GraphQLClient = GraphQLClient;
