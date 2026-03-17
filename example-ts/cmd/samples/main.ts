import { GraphQLClient, GraphQLErrors } from "gqlkit-ts";
import { QueryRoot } from "../../sdk/queries";
import { MutationRoot } from "../../sdk/mutations";

async function main() {
  console.log("TypeScript SDK Sample Queries");

  const client = new GraphQLClient("http://localhost:8081/query");

  const qr = new QueryRoot(client);
  const mr = new MutationRoot(client);

  try {
    await runPing(qr);
  } catch (err) {
    debugPrintError(err);
    process.exit(1);
  }

  try {
    await runEcho(qr);
  } catch (err) {
    debugPrintError(err);
  }

  try {
    await runSum(qr);
  } catch (err) {
    debugPrintError(err);
  }

  try {
    await runTodosConnection(qr);
  } catch (err) {
    debugPrintError(err);
  }

  try {
    await runUsers(qr);
  } catch (err) {
    debugPrintError(err);
  }

  try {
    await runCreateTodo(mr);
  } catch (err) {
    debugPrintError(err);
  }

  // Uncomment to run additional examples:
  // await runTodos(qr);
  // await runTodo(qr);
  // await runUser(qr);
  // await runDeleteTodo(mr);
  // await runCompleteAllTodos(mr);
}

function debugPrintError(err: unknown) {
  if (err instanceof GraphQLErrors) {
    console.error("GraphQL errors (raw):");
    console.error(JSON.stringify(err.errors, null, 2));
    return;
  }
  console.error("error:", err);
}

async function runPing(qr: QueryRoot) {
  const pingResult = await qr.ping().execute();
  console.log("Ping result:", pingResult);

  const rawData = await qr.ping().executeRaw();
  console.log("Ping raw data:", JSON.stringify(rawData, null, 2));
}

async function runEcho(qr: QueryRoot) {
  const echoResult = await qr.echo().message("Hello from SDK!").execute();
  console.log("Echo result:", echoResult);
}

async function runSum(qr: QueryRoot) {
  const sumResult = await qr.sum().a(10).b(20).execute();
  console.log("Sum result:", sumResult);
}

async function runTodosConnection(qr: QueryRoot) {
  const result = await qr
    .todosConnection()
    .filter({ done: false })
    .pagination({ limit: 10, offset: 0 })
    .select((conn) =>
      conn
        .totalCount()
        .edges((e) =>
          e
            .cursor()
            .node((t) =>
              t
                .id()
                .text()
                .done()
                .priority()
                .user((u) => u.id().name().email())
            )
        )
        .pageInfo((p) =>
          p.hasNextPage().hasPreviousPage().startCursor().endCursor()
        )
    )
    .execute();



  console.log("Todos connection:", JSON.stringify(result, null, 2));
}

async function runUsers(qr: QueryRoot) {
  const result = await qr
    .users()
    .select((u) => u.id().name().email().role())
    .execute();

  console.log("Users:", JSON.stringify(result, null, 2));
}

async function runCreateTodo(mr: MutationRoot) {
  const result = await mr
    .createTodo()
    .input({ text: "Buy milk", userId: "user-1" })
    .select((t) => t.id().text().done())
    .execute();

  console.log("Created todo:", JSON.stringify(result, null, 2));
}

async function runTodos(qr: QueryRoot) {
  const result = await qr
    .todos()
    .filter({ done: false })
    .select((t) => t.id().text().done().priority())
    .execute();

  console.log("Todos:", JSON.stringify(result, null, 2));
}

async function runTodo(qr: QueryRoot) {
  const result = await qr
    .todo()
    .id("todo-1")
    .select((t) => t.id().text().done().user((u) => u.id().name()))
    .execute();

  console.log("Todo:", JSON.stringify(result, null, 2));
}

async function runUser(qr: QueryRoot) {
  const result = await qr
    .user()
    .id("user-1")
    .select((u) => u.id().name().email().role())
    .execute();

  console.log("User:", JSON.stringify(result, null, 2));
}

async function runDeleteTodo(mr: MutationRoot) {
  const result = await mr.deleteTodo().id("todo-1").execute();
  console.log("Delete result:", result);
}

async function runCompleteAllTodos(mr: MutationRoot) {
  const result = await mr.completeAllTodos().execute();
  console.log("Completed count:", result);
}

main().catch(console.error);
