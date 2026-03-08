# @practor/generator

> TypeScript client code generator for Practor ORM.

Reads a parsed Practor schema (JSON) and generates a fully **type-safe TypeScript client** with model types, input types, filter types, and a client class with model delegates.

## Features

- 📝 **Model types** — `User`, `Post`, etc.
- 🔍 **Input types** — `UserCreateInput`, `UserWhereInput`, `UserUpdateInput`
- 🏗️ **Client class** — Extends `@practor/client` with typed model delegates
- 🎯 **Enum support** — First-class TypeScript enum generation
- ⚡ **Zero runtime** — Pure codegen, no runtime overhead

## Installation

```bash
npm install @practor/generator
```

## Usage

```typescript
import { generate, generateFromSchema } from "@practor/generator";

// From a schema file (uses the Go engine to parse)
await generateFromSchema("./schema.practor", "./generated/client");

// From a parsed schema object
generate(schemaJSON, "./generated/client");
```

## License

MIT
