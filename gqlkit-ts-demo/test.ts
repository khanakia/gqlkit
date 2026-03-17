import { GraphQLClient, GraphQLErrors, FieldSelection, BaseBuilder } from "gqlkit-ts";

// Verify all exports are importable
console.log("GraphQLClient:", typeof GraphQLClient);
console.log("GraphQLErrors:", typeof GraphQLErrors);
console.log("FieldSelection:", typeof FieldSelection);
console.log("BaseBuilder:", typeof BaseBuilder);

// Instantiate client
const client = new GraphQLClient("https://countries.trevorblades.com/graphql");

// Run a real query
async function main() {
  try {
    const data = await client.execute<{ countries: { name: string; code: string }[] }>(
      `query { countries { name code } }`
    );
    console.log(`\nFetched ${data.countries.length} countries. First 5:`);
    data.countries.slice(0, 5).forEach((c) => {
      console.log(`  ${c.code} - ${c.name}`);
    });
    console.log("\ngqlkit-ts is working!");
  } catch (err) {
    if (err instanceof GraphQLErrors) {
      console.error("GraphQL errors:", err.errors);
    } else {
      throw err;
    }
  }
}

main();
