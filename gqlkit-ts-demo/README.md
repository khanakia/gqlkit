# gqlkit-ts Demo

A minimal demo to verify the [gqlkit-ts](https://www.npmjs.com/package/gqlkit-ts) npm package works correctly.

## Setup

```bash
npm install
```

## Run

```bash
npm start
```

## What it does

- Imports all public exports from `gqlkit-ts` (`GraphQLClient`, `GraphQLErrors`, `FieldSelection`, `BaseBuilder`)
- Verifies each export is available
- Runs a live GraphQL query against a public Countries API
- Prints the first 5 countries from the response

## Expected output

```
GraphQLClient: function
GraphQLErrors: function
FieldSelection: function
BaseBuilder: function

Fetched 250 countries. First 5:
  AD - Andorra
  AE - United Arab Emirates
  AF - Afghanistan
  AG - Antigua and Barbuda
  AI - Anguilla

gqlkit-ts is working!
```
