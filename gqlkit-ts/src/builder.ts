/**
 * @module builder
 *
 * Provides the query-building primitives that generated SDK code extends.
 * - {@link FieldSelection} models the recursive field tree in a GraphQL selection set.
 * - {@link BaseBuilder} assembles a complete operation string (with variables) and executes it.
 */

import { GraphQLClient } from "./graphqlclient";

/**
 * Tracks which fields are selected in a GraphQL query.
 *
 * Maintains two lists:
 * - `fields`   : Scalar (leaf) field names like `id`, `name`.
 * - `children` : Nested object selections, each with its own {@link FieldSelection}.
 *
 * Used by generated builder classes to accumulate the selection set before
 * serializing it into a GraphQL query string via {@link build}.
 *
 * @example
 * ```ts
 * const sel = new FieldSelection();
 * sel.addField("id");
 * sel.addField("name");
 * const addressSel = new FieldSelection();
 * addressSel.addField("city");
 * sel.addChild("address", addressSel);
 * console.log(sel.build());
 * // =>
 * //   id
 * //   name
 * //   address {
 * //     city
 * //   }
 * ```
 */
export class FieldSelection {
  /** Scalar field names at this level of the selection. */
  private fields: string[] = [];

  /** Nested object fields, each mapping a field name to its sub-selection. */
  private children: Map<string, FieldSelection> = new Map();

  /**
   * Add a scalar (leaf) field to this selection level.
   * @param name - The field name to include (e.g., "id", "email").
   */
  addField(name: string): void {
    this.fields.push(name);
  }

  /**
   * Add a nested object field with its own sub-selection.
   * @param name  - The field name that returns an object type.
   * @param child - The {@link FieldSelection} describing which sub-fields to request.
   */
  addChild(name: string, child: FieldSelection): void {
    this.children.set(name, child);
  }

  /**
   * Serialize this selection into a GraphQL selection-set string.
   * Recursively builds nested selections with increasing indentation.
   *
   * @param indent - Number of leading spaces for the current depth (default: 2).
   * @returns A multi-line string representing the GraphQL field selection.
   */
  build(indent: number = 2): string {
    const pad = " ".repeat(indent);
    const parts: string[] = [];

    // Emit each scalar field on its own indented line
    for (const field of this.fields) {
      parts.push(`${pad}${field}`);
    }

    // Emit each nested field with its sub-selection wrapped in braces
    for (const [name, child] of this.children) {
      const nested = child.build(indent + 2);
      parts.push(`${pad}${name} {\n${nested}\n${pad}}`);
    }

    return parts.join("\n");
  }

  /**
   * Check whether this selection contains any fields at all.
   * An empty selection means no fields were requested at this level.
   * @returns `true` if no scalar fields and no nested children exist.
   */
  isEmpty(): boolean {
    return this.fields.length === 0 && this.children.size === 0;
  }
}

/**
 * Base class for generated GraphQL operation builders (queries and mutations).
 *
 * Generated SDK classes extend BaseBuilder to provide typed setter methods for
 * each argument and field-selection helpers. BaseBuilder handles:
 * - Storing operation arguments with their GraphQL type annotations.
 * - Accumulating the field selection via {@link FieldSelection}.
 * - Serializing everything into a valid GraphQL operation string.
 * - Executing the operation through the provided {@link GraphQLClient}.
 *
 * A typical generated builder calls `setArg()` for user-supplied arguments,
 * populates the selection via `getSelection()`, and finally calls `executeRaw()`
 * to send the request.
 *
 * @example
 * ```
 * // Generated code (simplified):
 * class GetUserBuilder extends BaseBuilder {
 *   constructor(client: GraphQLClient) {
 *     super(client, "query", "GetUser", "user");
 *   }
 *   id(value: string) { this.setArg("id", value, "ID!"); return this; }
 *   async exec() { return (await this.executeRaw()).user as User; }
 * }
 * ```
 */
export class BaseBuilder {
  /** The GraphQL client used to execute the built operation. */
  private client: GraphQLClient;

  /** The operation type keyword: "query" or "mutation". */
  private opType: string;

  /** The operation name used in the GraphQL document (e.g., "GetUser"). */
  private opName: string;

  /** The root field name inside the operation (e.g., "user", "createUser"). */
  private fieldName: string;

  /**
   * Map of argument name to its runtime value and GraphQL type string.
   * For example: "id" -> { value: "123", graphqlType: "ID!" }
   */
  private args: Map<string, { value: unknown; graphqlType: string }> =
    new Map();

  /** The field selection tree for the operation's return type. */
  private selection: FieldSelection = new FieldSelection();

  /**
   * @param client    - The GraphQL client to execute operations with.
   * @param opType    - Operation kind: "query" or "mutation".
   * @param opName    - The named operation identifier (e.g., "GetUser").
   * @param fieldName - The root field to query/mutate (e.g., "user").
   */
  constructor(
    client: GraphQLClient,
    opType: string,
    opName: string,
    fieldName: string
  ) {
    this.client = client;
    this.opType = opType;
    this.opName = opName;
    this.fieldName = fieldName;
  }

  /**
   * Register an argument for the operation.
   * Called by generated setter methods to record each argument's value and type.
   *
   * @param name        - The argument name (doubles as the variable name).
   * @param value       - The runtime value to send as a variable.
   * @param graphqlType - The GraphQL type annotation (e.g., "String!", "Int", "[ID!]!").
   */
  setArg(name: string, value: unknown, graphqlType: string): void {
    this.args.set(name, { value, graphqlType });
  }

  /**
   * Access the field selection tree so callers can add fields and nested selections.
   * @returns The root {@link FieldSelection} for this operation.
   */
  getSelection(): FieldSelection {
    return this.selection;
  }

  /**
   * Access the underlying GraphQL client (used by generated code if needed).
   * @returns The {@link GraphQLClient} instance.
   */
  getClient(): GraphQLClient {
    return this.client;
  }

  /**
   * Extract the runtime variable values as a plain object for the request payload.
   * Maps each registered argument name to its value, discarding the type info.
   *
   * @returns A `{ [argName]: value }` record suitable for the `variables` field.
   */
  getVariables(): Record<string, unknown> {
    const vars: Record<string, unknown> = {};
    for (const [name, { value }] of this.args) {
      vars[name] = value;
    }
    return vars;
  }

  /**
   * Build the complete GraphQL operation string.
   *
   * Produces output like:
   * ```graphql
   * query GetUser($id: ID!) {
   *   user(id: $id) {
   *     id
   *     name
   *   }
   * }
   * ```
   *
   * Steps:
   * 1. Build variable declarations from registered args (e.g., `$id: ID!`).
   * 2. Build argument pass-throughs for the root field (e.g., `id: $id`).
   * 3. Serialize the field selection set (if any fields were selected).
   * 4. Combine into the final operation string.
   *
   * @returns The full GraphQL operation document as a string.
   */
  buildQuery(): string {
    // Collect variable declarations (e.g., "$id: ID!") and argument references (e.g., "id: $id")
    const varDecls: string[] = [];
    const argPasses: string[] = [];

    for (const [name, { graphqlType }] of this.args) {
      varDecls.push(`$${name}: ${graphqlType}`);
      argPasses.push(`${name}: $${name}`);
    }

    // Format the variable declaration and argument lists (empty string if no args)
    const varStr = varDecls.length > 0 ? `(${varDecls.join(", ")})` : "";
    const argStr = argPasses.length > 0 ? `(${argPasses.join(", ")})` : "";

    // Build the selection set body; omit braces entirely if no fields selected
    const selStr = this.selection.isEmpty()
      ? ""
      : ` {\n${this.selection.build(4)}\n  }`;

    // Assemble: "query GetUser($id: ID!) {\n  user(id: $id) {\n    ...\n  }\n}"
    return `${this.opType} ${this.opName}${varStr} {\n  ${this.fieldName}${argStr}${selStr}\n}`;
  }

  /**
   * Build and execute the operation, returning the raw `data` object.
   *
   * Generated subclasses typically wrap this method to extract and cast
   * the specific root field from the response (e.g., `(await executeRaw()).user`).
   *
   * @returns The full `data` object from the GraphQL response.
   * @throws {GraphQLErrors} If the server returns errors.
   */
  async executeRaw(): Promise<Record<string, unknown>> {
    const query = this.buildQuery();
    const variables = this.getVariables();
    return await this.client.execute<Record<string, unknown>>(
      query,
      variables
    );
  }
}
