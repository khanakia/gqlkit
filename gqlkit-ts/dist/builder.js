"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BaseBuilder = exports.FieldSelection = void 0;
/** FieldSelection tracks which fields are selected in a GraphQL query */
class FieldSelection {
    constructor() {
        this.fields = [];
        this.children = new Map();
    }
    /** Add a scalar field to the selection */
    addField(name) {
        this.fields.push(name);
    }
    /** Add a nested field with its own selection */
    addChild(name, child) {
        this.children.set(name, child);
    }
    /** Build the GraphQL field selection string */
    build(indent = 2) {
        const pad = " ".repeat(indent);
        const parts = [];
        for (const field of this.fields) {
            parts.push(`${pad}${field}`);
        }
        for (const [name, child] of this.children) {
            const nested = child.build(indent + 2);
            parts.push(`${pad}${name} {\n${nested}\n${pad}}`);
        }
        return parts.join("\n");
    }
    /** Check if the selection is empty */
    isEmpty() {
        return this.fields.length === 0 && this.children.size === 0;
    }
}
exports.FieldSelection = FieldSelection;
/** BaseBuilder provides common functionality for query/mutation builders */
class BaseBuilder {
    constructor(client, opType, opName, fieldName) {
        this.args = new Map();
        this.selection = new FieldSelection();
        this.client = client;
        this.opType = opType;
        this.opName = opName;
        this.fieldName = fieldName;
    }
    /** Set an argument for the operation */
    setArg(name, value, graphqlType) {
        this.args.set(name, { value, graphqlType });
    }
    /** Get the field selection */
    getSelection() {
        return this.selection;
    }
    /** Get the GraphQL client */
    getClient() {
        return this.client;
    }
    /** Get the variables map */
    getVariables() {
        const vars = {};
        for (const [name, { value }] of this.args) {
            vars[name] = value;
        }
        return vars;
    }
    /** Build the GraphQL query string */
    buildQuery() {
        // Build variable declarations
        const varDecls = [];
        const argPasses = [];
        for (const [name, { graphqlType }] of this.args) {
            varDecls.push(`$${name}: ${graphqlType}`);
            argPasses.push(`${name}: $${name}`);
        }
        const varStr = varDecls.length > 0 ? `(${varDecls.join(", ")})` : "";
        const argStr = argPasses.length > 0 ? `(${argPasses.join(", ")})` : "";
        const selStr = this.selection.isEmpty()
            ? ""
            : ` {\n${this.selection.build(4)}\n  }`;
        return `${this.opType} ${this.opName}${varStr} {\n  ${this.fieldName}${argStr}${selStr}\n}`;
    }
    /** Execute the operation and return raw response */
    async executeRaw() {
        const query = this.buildQuery();
        const variables = this.getVariables();
        return await this.client.execute(query, variables);
    }
}
exports.BaseBuilder = BaseBuilder;
