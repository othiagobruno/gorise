const test = require("node:test");
const assert = require("node:assert/strict");

const { PractorClient, PractorError, sql } = require("../dist");

test("safe raw SQL compiles placeholders and rejects string overloads", async () => {
  const calls = [];
  const client = new PractorClient();

  client.connected = true;
  client.engine = {
    request: async (method, payload) => {
      calls.push({ method, payload });
      if (method === "db.executeRaw") {
        return { count: 2 };
      }
      return [{ id: 1 }];
    },
  };

  const rows = await client.$queryRaw`SELECT * FROM users WHERE id = ${42}`;
  const count = await client.$executeRaw(
    sql`UPDATE users SET active = ${true} WHERE ${sql`email = ${"a@b.com"}`}`,
  );
  const unsafeRows = await client.$queryRawUnsafe(
    "SELECT * FROM users WHERE email = $1",
    "x@y.com",
  );

  await assert.rejects(
    () => client.$queryRaw("SELECT * FROM users"),
    (error) =>
      error instanceof PractorError &&
      error.message.includes("$queryRawUnsafe"),
  );

  await assert.rejects(
    () => client.$queryRaw(sql`SELECT 1`, 123),
    (error) =>
      error instanceof PractorError &&
      error.message.includes("prebuilt sql"),
  );

  assert.equal(rows.length, 1);
  assert.equal(count, 2);
  assert.equal(unsafeRows.length, 1);
  assert.equal(calls[0].payload.query, "SELECT * FROM users WHERE id = $1");
  assert.equal(
    calls[1].payload.query,
    "UPDATE users SET active = $1 WHERE email = $2",
  );
  assert.equal(
    calls[2].payload.query,
    "SELECT * FROM users WHERE email = $1",
  );
});
