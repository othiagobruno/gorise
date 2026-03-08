/**
 * @practor/client — PractorClient base class
 *
 * Provides the Prisma-compatible API: model delegates, transactions,
 * raw queries, and lifecycle management.
 */

import { PractorEngine, PractorError } from "./engine";
import {
  MiddlewareEngine,
  type MiddlewareFunction,
  type MiddlewareParams,
} from "./middleware";
import type {
  PractorClientOptions,
  ModelDelegate,
  TransactionOptions,
  PaginationResult,
  CursorPaginationResult,
  PoolStats,
} from "./types";

/**
 * A deferred query descriptor captured during batch $transaction.
 *
 * Why? In batch mode we can't execute promises eagerly — we need to
 * collect the operation descriptors and replay them inside the TX.
 */
interface QueryDescriptor {
  model: string;
  action: string;
  args: Record<string, unknown>;
  method: "query" | "mutation";
}

interface DelegateOperation {
  methodName: string;
  action: string;
}

const QUERY_ACTIONS = new Set([
  "findMany",
  "findUnique",
  "findFirst",
  "findUniqueOrThrow",
  "findFirstOrThrow",
  "count",
  "aggregate",
  "groupBy",
  "findManyPaginated",
  "findManyCursorPaginated",
]);

const DELEGATE_OPERATIONS: DelegateOperation[] = [
  { methodName: "findMany", action: "findMany" },
  { methodName: "findUnique", action: "findUnique" },
  { methodName: "findFirst", action: "findFirst" },
  { methodName: "findUniqueOrThrow", action: "findUniqueOrThrow" },
  { methodName: "findFirstOrThrow", action: "findFirstOrThrow" },
  { methodName: "create", action: "create" },
  { methodName: "createMany", action: "createMany" },
  { methodName: "update", action: "update" },
  { methodName: "updateMany", action: "updateMany" },
  { methodName: "delete", action: "delete" },
  { methodName: "deleteMany", action: "deleteMany" },
  { methodName: "upsert", action: "upsert" },
  { methodName: "count", action: "count" },
  { methodName: "aggregate", action: "aggregate" },
  { methodName: "groupBy", action: "groupBy" },
  { methodName: "paginate", action: "findManyPaginated" },
  { methodName: "cursorPaginate", action: "findManyCursorPaginated" },
];

function isQueryAction(action: string): boolean {
  return QUERY_ACTIONS.has(action);
}

function cloneArgs(args: Record<string, unknown>): Record<string, unknown> {
  try {
    return structuredClone(args);
  } catch {
    return { ...args };
  }
}

class PractorPromise<T> implements Promise<T> {
  readonly [Symbol.toStringTag] = "Promise";

  private promise: Promise<T> | null = null;

  constructor(
    private readonly executor: () => Promise<T>,
    readonly descriptor?: QueryDescriptor,
  ) {}

  get started(): boolean {
    return this.promise !== null;
  }

  private getOrCreatePromise(): Promise<T> {
    if (this.promise === null) {
      this.promise = this.executor();
    }

    return this.promise;
  }

  then<TResult1 = T, TResult2 = never>(
    onfulfilled?:
      | ((value: T) => TResult1 | PromiseLike<TResult1>)
      | null,
    onrejected?:
      | ((reason: any) => TResult2 | PromiseLike<TResult2>)
      | null,
  ): Promise<TResult1 | TResult2> {
    return this.getOrCreatePromise().then(onfulfilled, onrejected);
  }

  catch<TResult = never>(
    onrejected?:
      | ((reason: any) => TResult | PromiseLike<TResult>)
      | null,
  ): Promise<T | TResult> {
    return this.getOrCreatePromise().catch(onrejected);
  }

  finally(onfinally?: (() => void) | null): Promise<T> {
    return this.getOrCreatePromise().finally(onfinally ?? undefined);
  }
}

function isPractorPromise(value: unknown): value is PractorPromise<unknown> {
  return value instanceof PractorPromise;
}

/**
 * PractorClient is the main entry point for database operations.
 *
 * Usage:
 * ```typescript
 * const practor = new PractorClient();
 * await practor.$connect();
 *
 * const users = await practor.user.findMany();
 * await practor.$disconnect();
 * ```
 */
export class PractorClient {
  private engine: PractorEngine;
  private options: PractorClientOptions;
  private connected = false;
  private middlewareEngine: MiddlewareEngine = new MiddlewareEngine();
  private modelDelegates: Map<string, ModelDelegate> = new Map();

  /** Known model names from the schema. Populated after connect. */
  private modelNames: string[] = [];

  [key: string]: any;

  constructor(options: PractorClientOptions = {}) {
    this.options = options;
    this.engine = new PractorEngine({
      enginePath: options.enginePath,
      schemaPath: options.schemaPath,
      datasourceUrl: options.datasourceUrl,
      poolConfig: options.pool,
    });

    // Set up logging
    if (options.log) {
      this.engine.on("log", (msg: string) => {
        if (options.log!.includes("info")) {
          console.log(`[Practor] ${msg}`);
        }
      });
    }
  }

  /**
   * Connects to the database by starting the engine process.
   * Also fetches the schema to create model delegates.
   */
  async $connect(): Promise<void> {
    if (this.connected) return;

    await this.engine.start();

    // Fetch schema metadata to create model delegates
    try {
      const schemaResult = (await this.engine.request(
        "schema.getJSON",
        {},
      )) as any;
      if (schemaResult && schemaResult.models) {
        for (const model of schemaResult.models) {
          const name = model.name;
          const camelName = name.charAt(0).toLowerCase() + name.slice(1);
          const delegate = this.createModelDelegate(name);
          this.modelDelegates.set(camelName, delegate);
          this.modelNames.push(name);

          // Make delegate accessible as client.user, client.post, etc.
          (this as any)[camelName] = delegate;
        }
      }
    } catch (err) {
      // Schema fetch is optional — delegates can be created lazily
      if (this.options.log?.includes("warn")) {
        console.warn("[Practor] Failed to fetch schema:", err);
      }
    }

    this.connected = true;
  }

  /** Disconnects from the database. */
  async $disconnect(): Promise<void> {
    if (!this.connected) return;
    await this.engine.stop();
    this.connected = false;
  }

  /**
   * Returns runtime connection pool statistics from the Go engine.
   *
   * @example
   * ```ts
   * const stats = await practor.$pool();
   * console.log(`Active: ${stats.inUse}, Idle: ${stats.idle}`);
   * ```
   */
  async $pool(): Promise<PoolStats> {
    this.ensureConnected();
    const result = await this.engine.request("pool.getStats", {});
    return result as PoolStats;
  }

  /**
   * Registers a middleware function that intercepts all model operations.
   *
   * Middleware runs in FIFO order — first registered = outermost wrapper.
   * Each middleware can inspect/mutate params and results.
   *
   * @example
   * ```ts
   * // Logging middleware
   * practor.$use(async (params, next) => {
   *   console.log(`${params.model}.${params.action}`);
   *   const result = await next(params);
   *   console.log(`Done in ${Date.now() - start}ms`);
   *   return result;
   * });
   *
   * // Soft-delete middleware
   * practor.$use(async (params, next) => {
   *   if (params.action === 'delete') {
   *     params.action = 'update';
   *     params.args = { ...params.args, data: { deletedAt: new Date() } };
   *   }
   *   return next(params);
   * });
   * ```
   */
  $use(fn: MiddlewareFunction): void {
    this.middlewareEngine.use(fn);
  }

  /**
   * Executes a raw SQL query that does not return data (INSERT, UPDATE, DELETE).
   *
   * @example
   * ```ts
   * const count = await practor.$executeRaw`UPDATE users SET active = true WHERE id = ${1}`;
   * ```
   */
  async $executeRaw(
    query: string | TemplateStringsArray,
    ...values: unknown[]
  ): Promise<number> {
    this.ensureConnected();
    const { sql, args } = this.processRawQuery(query, values);
    const result = (await this.engine.request("db.executeRaw", {
      query: sql,
      args,
    })) as any;
    return result.count ?? 0;
  }

  /**
   * Executes a raw SQL query that returns rows.
   *
   * @example
   * ```ts
   * const users = await practor.$queryRaw`SELECT * FROM users WHERE age > ${18}`;
   * ```
   */
  async $queryRaw<T = unknown>(
    query: string | TemplateStringsArray,
    ...values: unknown[]
  ): Promise<T[]> {
    this.ensureConnected();
    const { sql, args } = this.processRawQuery(query, values);
    const result = await this.engine.request("db.queryRaw", {
      query: sql,
      args,
    });
    return (result as T[]) ?? [];
  }

  /**
   * Executes operations in a database transaction.
   *
   * @example
   * ```ts
   * // Interactive transaction
   * await practor.$transaction(async (tx) => {
   *   const user = await tx.user.create({ data: { email: 'a@b.com' } });
   *   await tx.post.create({ data: { title: 'Hello', authorId: user.id } });
   * });
   *
   * // Batch transaction
   * const [user, post] = await practor.$transaction([
   *   practor.user.create({ data: { email: 'a@b.com' } }),
   *   practor.post.create({ data: { title: 'Hello', authorId: 1 } }),
   * ]);
   * ```
   */
  async $transaction<T>(
    arg: ((tx: PractorClient) => Promise<T>) | Promise<unknown>[],
    options?: TransactionOptions,
  ): Promise<T | unknown[]> {
    this.ensureConnected();

    if (Array.isArray(arg)) {
      return this.executeBatchTransaction(arg, options);
    }

    return this.executeInteractiveTransaction(arg, options);
  }

  // ============================================================================
  // Transaction internals
  // ============================================================================

  /**
   * Batch transaction: resolves an array of promises within a single TX.
   *
   * Why deferred descriptors? The user passes `practor.user.create(...)` which
   * returns a Promise. We resolve them — if any fail we rollback the whole TX.
   */
  private async executeBatchTransaction(
    operations: Promise<unknown>[],
    options?: TransactionOptions,
  ): Promise<unknown[]> {
    const descriptors = operations.map((operation, index) =>
      this.getBatchDescriptor(operation, index),
    );

    // Begin the transaction on the engine
    const beginResult = (await this.engine.request("transaction.begin", {
      isolationLevel: options?.isolationLevel ?? "",
      timeout: options?.timeout ?? 0,
    })) as { txId: string };

    const txId = beginResult.txId;

    try {
      const results: unknown[] = [];
      for (const descriptor of descriptors) {
        results.push(await this.executeOperation(descriptor, txId));
      }

      await this.engine.request("transaction.commit", { txId });
      return results;
    } catch (error) {
      await this.engine
        .request("transaction.rollback", { txId })
        .catch(() => {}); // Swallow rollback errors
      throw error;
    }
  }

  /**
   * Interactive transaction: provides a transactional client proxy.
   *
   * The callback receives a `PractorClient`-like object where every model
   * delegate routes queries through `transaction.query` / `transaction.mutation`
   * using the active txId, ensuring all operations share the same SQL TX.
   */
  private async executeInteractiveTransaction<T>(
    fn: (tx: PractorClient) => Promise<T>,
    options?: TransactionOptions,
  ): Promise<T> {
    const beginResult = (await this.engine.request("transaction.begin", {
      isolationLevel: options?.isolationLevel ?? "",
      timeout: options?.timeout ?? 0,
    })) as { txId: string };

    const txId = beginResult.txId;

    // Build a transaction-scoped client proxy
    const txClient = this.createTransactionProxy(txId);

    try {
      const result = await fn(txClient);
      await this.engine.request("transaction.commit", { txId });
      return result;
    } catch (error) {
      await this.engine
        .request("transaction.rollback", { txId })
        .catch(() => {});
      throw error;
    }
  }

  /**
   * Creates a proxy PractorClient that routes all model operations through
   * the engine's transaction.query / transaction.mutation methods.
   */
  private createTransactionProxy(txId: string): PractorClient {
    const proxy = Object.create(this) as PractorClient;

    for (const [camelName, _delegate] of this.modelDelegates) {
      const modelName = camelName.charAt(0).toUpperCase() + camelName.slice(1);
      const txDelegate: Record<string, Function> = {};
      for (const operation of DELEGATE_OPERATIONS) {
        txDelegate[operation.methodName] = (
          args: Record<string, unknown> = {},
        ) => this.createDelegateCall(modelName, operation, args, txId, false);
      }

      (proxy as any)[camelName] = txDelegate;
    }

    return proxy;
  }

  // ============================================================================
  // Model delegate factory
  // ============================================================================

  /**
   * Creates a model delegate that proxies all CRUD operations to the engine.
   *
   * Why a Proxy? This allows intercepting all method calls for logging,
   * middleware, and lazy query building.
   */
  private createModelDelegate(modelName: string): ModelDelegate {
    const delegate: Record<string, Function> = {};

    for (const operation of DELEGATE_OPERATIONS) {
      delegate[operation.methodName] = (
        args: Record<string, unknown> = {},
      ) => this.createDelegateCall(modelName, operation, args);
    }

    return delegate as unknown as ModelDelegate;
  }

  private createDelegateCall<T>(
    modelName: string,
    operation: DelegateOperation,
    args: Record<string, unknown>,
    txId?: string,
    captureDescriptor = true,
  ): Promise<T> {
    const descriptor: QueryDescriptor = {
      model: modelName,
      action: operation.action,
      args,
      method: isQueryAction(operation.action) ? "query" : "mutation",
    };

    return new PractorPromise(async () => {
      this.ensureConnected();

      if (this.options.log?.includes("query")) {
        console.log(
          `[Practor Query] ${modelName}.${operation.methodName}`,
          JSON.stringify(args, null, 2),
        );
      }

      return this.executeOperation<T>(descriptor, txId);
    }, captureDescriptor ? descriptor : undefined) as Promise<T>;
  }

  private async executeOperation<T>(
    params: QueryDescriptor,
    txId?: string,
  ): Promise<T> {
    return (await this.middlewareEngine.execute(
      params,
      async (p: MiddlewareParams) => {
        const rpcMethod = txId
          ? p.method === "query"
            ? "transaction.query"
            : "transaction.mutation"
          : p.method;
        const rpcParams = txId
          ? {
              txId,
              model: p.model,
              action: p.action,
              args: p.args,
            }
          : {
              model: p.model,
              action: p.action,
              args: p.args,
            };
        const result = await this.engine.request(rpcMethod, rpcParams);
        const response = result as any;
        return response?.data ?? response;
      },
    )) as T;
  }

  private getBatchDescriptor(
    operation: Promise<unknown>,
    index: number,
  ): QueryDescriptor {
    if (!isPractorPromise(operation) || !operation.descriptor) {
      throw new PractorError(
        `Batch transaction operation at index ${index} is not a direct Practor query. Pass delegate calls to $transaction([...]) without awaiting them first.`,
        -1,
      );
    }

    if (operation.started) {
      throw new PractorError(
        `Batch transaction operation at index ${index} was already started. Pass untouched Practor query objects to $transaction([...]).`,
        -1,
      );
    }

    return {
      ...operation.descriptor,
      args: cloneArgs(operation.descriptor.args),
    };
  }

  // ============================================================================
  // Utilities
  // ============================================================================

  /** Ensures the client is connected. */
  private ensureConnected(): void {
    if (!this.connected) {
      throw new PractorError(
        "PractorClient is not connected. Call $connect() first.",
        -1,
      );
    }
  }

  /**
   * Processes raw SQL queries. Supports tagged template literals.
   *
   * Why? Tagged templates provide SQL injection safety by parameterizing values.
   */
  private processRawQuery(
    query: string | TemplateStringsArray,
    values: unknown[],
  ): { sql: string; args: unknown[] } {
    if (typeof query === "string") {
      return { sql: query, args: values };
    }

    // Tagged template literal: SELECT * FROM users WHERE id = ${id}
    let sql = "";
    const args: unknown[] = [];
    let paramIndex = 0;

    for (let i = 0; i < query.length; i++) {
      sql += query[i];
      if (i < values.length) {
        paramIndex++;
        sql += `$${paramIndex}`;
        args.push(values[i]);
      }
    }

    return { sql, args };
  }
}
