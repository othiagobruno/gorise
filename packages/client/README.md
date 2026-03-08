# @practor/client

> Prisma-compatible ORM client powered by a Go query engine.

Practor is a **high-performance ORM for Node.js** with its query engine written in **Go**. It provides a Prisma-like developer experience with the raw speed of a compiled language.

## Features

- 🚀 **Go-powered query engine** — Compiled binary for maximum throughput
- 🔒 **Type-safe API** — Full TypeScript type generation from your schema
- 🔄 **Prisma-compatible** — Familiar `findMany`, `create`, `update`, `delete` API
- 💳 **Transactions** — Interactive and batch transaction support
- 📄 **Pagination** — Built-in paginated queries with metadata
- 📡 **JSON-RPC** — Clean process isolation via stdin/stdout IPC

## Installation

```bash
npm install @practor/client
```

## Quick Start

```typescript
import { PractorClient } from "@practor/client";

const practor = new PractorClient({
  datasourceUrl: process.env.DATABASE_URL,
});

await practor.$connect();

// Find many
const users = await practor.user.findMany({
  where: { active: true },
  orderBy: { createdAt: "desc" },
  take: 10,
});

// Create
const newUser = await practor.user.create({
  data: { email: "hello@practor.dev", name: "Practor" },
});

// Transactions
await practor.$transaction(async (tx) => {
  const user = await tx.user.create({ data: { email: "tx@practor.dev" } });
  await tx.post.create({ data: { title: "Hello", authorId: user.id } });
});

await practor.$disconnect();
```

## License

MIT
