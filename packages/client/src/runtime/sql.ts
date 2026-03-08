/**
 * Utilities for building parameterized raw SQL queries safely.
 */

export class Sql {
  readonly strings: readonly string[];
  readonly values: readonly unknown[];

  constructor(strings: readonly string[], values: readonly unknown[]) {
    this.strings = [...strings];
    this.values = [...values];
  }
}

export type SafeSqlQuery = TemplateStringsArray | Sql;

/** Creates a reusable parameterized SQL fragment. */
export function sql(
  strings: TemplateStringsArray,
  ...values: unknown[]
): Sql {
  return new Sql(Array.from(strings), values);
}

export function isSql(value: unknown): value is Sql {
  return value instanceof Sql;
}

/** Compiles a tagged-template query into PostgreSQL positional parameters. */
export function compileSafeSql(
  query: SafeSqlQuery,
  values: readonly unknown[] = [],
): { sql: string; args: unknown[] } {
  if (isSql(query)) {
    return compileSqlParts(query.strings, query.values, 0);
  }

  return compileSqlParts(Array.from(query), values, 0);
}

function compileSqlParts(
  strings: readonly string[],
  values: readonly unknown[],
  paramOffset: number,
): { sql: string; args: unknown[] } {
  let sqlText = "";
  const args: unknown[] = [];
  let paramIndex = paramOffset;

  for (let i = 0; i < strings.length; i++) {
    sqlText += strings[i];

    if (i >= values.length) {
      continue;
    }

    const value = values[i];
    if (isSql(value)) {
      const nested = compileSqlParts(value.strings, value.values, paramIndex);
      sqlText += nested.sql;
      args.push(...nested.args);
      paramIndex += nested.args.length;
      continue;
    }

    paramIndex++;
    sqlText += `$${paramIndex}`;
    args.push(value);
  }

  return { sql: sqlText, args };
}
