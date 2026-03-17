import { GraphQLClient } from "./graphqlclient";
/** FieldSelection tracks which fields are selected in a GraphQL query */
export declare class FieldSelection {
    private fields;
    private children;
    /** Add a scalar field to the selection */
    addField(name: string): void;
    /** Add a nested field with its own selection */
    addChild(name: string, child: FieldSelection): void;
    /** Build the GraphQL field selection string */
    build(indent?: number): string;
    /** Check if the selection is empty */
    isEmpty(): boolean;
}
/** BaseBuilder provides common functionality for query/mutation builders */
export declare class BaseBuilder {
    private client;
    private opType;
    private opName;
    private fieldName;
    private args;
    private selection;
    constructor(client: GraphQLClient, opType: string, opName: string, fieldName: string);
    /** Set an argument for the operation */
    setArg(name: string, value: unknown, graphqlType: string): void;
    /** Get the field selection */
    getSelection(): FieldSelection;
    /** Get the GraphQL client */
    getClient(): GraphQLClient;
    /** Get the variables map */
    getVariables(): Record<string, unknown>;
    /** Build the GraphQL query string */
    buildQuery(): string;
    /** Execute the operation and return raw response */
    executeRaw(): Promise<Record<string, unknown>>;
}
