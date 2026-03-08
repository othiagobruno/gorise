# Practor ORM — Agent Context

## Project Identity

- **Name**: Practor
- **Scope**: `@practor` on npm
- **Repository**: `https://github.com/othiagobruno/practor`
- **License**: MIT
- **Schema extension**: `.practor`

## Architecture

Practor is a **Prisma-compatible ORM for Node.js** powered by a **Go query engine**. The engine runs as a compiled binary sidecar, communicating via **JSON-RPC 2.0 over stdin/stdout**.

```
Node.js App  ←─ JSON-RPC 2.0 (stdin/stdout) ─→  Go Engine  ←─ SQL ─→  PostgreSQL
```

## Monorepo Structure

```
practor/
├── engine/                          # Go Query Engine
│   ├── cmd/practor/main.go          # Entrypoint & JSON-RPC server
│   ├── go.mod                       # Module: github.com/practor/practor-engine
│   └── internal/
│       ├── schema/                  # PSL parser
│       ├── query/                   # Query builder & executor
│       ├── connector/               # Database connectors (PostgreSQL)
│       ├── migration/               # Migration engine
│       └── protocol/               # JSON-RPC types & handlers
├── packages/
│   ├── client/                      # @practor/client (v0.1.0)
│   │   └── src/
│   │       ├── index.ts             # Public exports
│   │       └── runtime/
│   │           ├── client.ts        # PractorClient class
│   │           ├── engine.ts        # Engine process manager
│   │           ├── middleware.ts     # $use() middleware pipeline
│   │           └── types.ts         # TypeScript type definitions
│   ├── generator/                   # @practor/generator (v0.1.0)
│   │   └── src/
│   │       ├── index.ts
│   │       ├── generator.ts         # Code generation logic
│   │       └── templates/           # Output templates
│   └── cli/                         # @practor/cli (v0.1.0)
│       └── src/
│           ├── index.ts             # CLI entrypoint (bin: "practor")
│           └── commands/            # CLI command handlers
├── bin/                             # Compiled engine binary output
├── schema.practor                   # Example/dev schema
├── tsconfig.base.json               # Shared TS config (ES2022, strict)
└── package.json                     # Monorepo root (npm workspaces)
```

## Tech Stack

| Layer     | Technology                        |
| --------- | --------------------------------- |
| Engine    | Go 1.25+, `lib/pq`, `google/uuid` |
| Runtime   | TypeScript (strict), Node.js ≥ 18 |
| IPC       | JSON-RPC 2.0 over stdin/stdout    |
| Database  | PostgreSQL                        |
| Build     | `tsc` (per workspace)             |
| Packaging | npm workspaces, `@practor` scope  |

## Build Commands

```bash
# Install dependencies
npm install

# Build Go engine binary → bin/practor-engine
npm run build:engine

# Build all TypeScript packages
npm run build

# Build individual package
cd packages/client && npm run build
```

## NPM Publication

### Package Versions

All packages are at `0.1.0`. Version bumps must be synchronized.

### Publication Order (dependency chain)

1. `@practor/client` — base types and runtime (no internal deps)
2. `@practor/generator` — code generation (no internal deps)
3. `@practor/cli` — depends on `@practor/generator`

### Publish Sequence

```bash
# 1. Clean build
npm run build

# 2. Dry-run each package
cd packages/client && npm pack --dry-run
cd packages/generator && npm pack --dry-run
cd packages/cli && npm pack --dry-run

# 3. Publish in order
cd packages/client && npm publish --access public
cd packages/generator && npm publish --access public
cd packages/cli && npm publish --access public
```

### Auth via Token

```bash
npm config set //registry.npmjs.org/:_authToken <TOKEN>
```

> If 2FA is active and token doesn't bypass it, use `--otp <CODE>` on each publish.

### Published Files

Each package ships only `dist/` and `README.md` (configured via `"files"` in package.json).

## Key Features Implemented

- ✅ CRUD: `findMany`, `findUnique`, `findFirst`, `create`, `update`, `delete`, `upsert`
- ✅ Transactions: interactive callbacks + batch arrays
- ✅ Offset-based pagination (`paginate()`)
- ✅ Cursor-based pagination (`cursorPaginate()`)
- ✅ Relation queries: `include` and `select` with nested loading
- ✅ Middleware/hooks: `$use()` with FIFO pipeline
- ✅ Connection pooling: configurable via options or env vars, `$pool()` stats
- ✅ Raw SQL: `$queryRaw`, `$executeRaw`
- ✅ Migrations: `migrate dev` and `migrate deploy`
- ✅ CLI: `init`, `generate`, `validate`, `db push`

## CLI Commands

| Command                  | Description                              |
| ------------------------ | ---------------------------------------- |
| `practor init`           | Initialize a new project with schema     |
| `practor generate`       | Generate TypeScript client from schema   |
| `practor validate`       | Validate schema syntax                   |
| `practor db push`        | Push schema changes to database          |
| `practor migrate dev`    | Create and apply migration (development) |
| `practor migrate deploy` | Apply pending migrations (production)    |

## Naming Rules

- **Project name**: Practor (capital P)
- **npm scope**: `@practor`
- **Schema files**: `schema.practor` or `*.practor`
- **Engine binary**: `practor-engine`
- **CLI binary**: `practor`
- **Environment prefix**: `PRACTOR_`
