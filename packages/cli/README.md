# @practor/cli

> Command-line interface for Practor ORM — schema management, code generation, and database operations.

## Installation

```bash
npm install -g @practor/cli
# or use npx
npx @practor/cli init
```

## Commands

| Command               | Description                                |
| --------------------- | ------------------------------------------ |
| `practor init`        | Initialize a new Practor project           |
| `practor generate`    | Generate the TypeScript client from schema |
| `practor validate`    | Validate the schema file                   |
| `practor db push`     | Push schema changes to the database        |
| `practor migrate dev` | Create and apply a migration               |

## Quick Start

```bash
# Initialize project
npx practor init

# Edit your schema.practor, then generate the client
npx practor generate

# Push schema to database
npx practor db push
```

## License

MIT
