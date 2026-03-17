/**
 * @module gqlkit-ts
 *
 * Public API surface for the gqlkit-ts runtime library.
 * Generated GraphQL SDKs depend on these exports to build and execute queries.
 *
 * Re-exports:
 * - {@link FieldSelection} - Tracks which fields are selected in a GraphQL query tree.
 * - {@link BaseBuilder}    - Base class for generated operation builders (query/mutation).
 * - {@link GraphQLClient}  - HTTP client that sends GraphQL operations to a server.
 * - {@link GraphQLErrors}  - Error class thrown when the server returns GraphQL errors.
 * - {@link ClientOptions}  - Configuration options for GraphQLClient (headers, auth, custom fetch).
 */

export { FieldSelection } from "./builder";
export { BaseBuilder } from "./builder";
export { GraphQLClient } from "./graphqlclient";
export { GraphQLErrors } from "./graphqlclient";
export type { ClientOptions } from "./graphqlclient";
