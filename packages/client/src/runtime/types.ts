/**
 * @practor/client — Core runtime types
 *
 * Type-safe query argument definitions matching Prisma's API surface.
 */

// ============================================================================
// Scalar filters
// ============================================================================

/** String filter operators. */
export interface StringFilter {
  equals?: string | null;
  not?: string | StringFilter | null;
  in?: string[];
  notIn?: string[];
  lt?: string;
  lte?: string;
  gt?: string;
  gte?: string;
  contains?: string;
  startsWith?: string;
  endsWith?: string;
  mode?: "default" | "insensitive";
}

/** Integer filter operators. */
export interface IntFilter {
  equals?: number | null;
  not?: number | IntFilter | null;
  in?: number[];
  notIn?: number[];
  lt?: number;
  lte?: number;
  gt?: number;
  gte?: number;
}

/** Float filter operators. */
export interface FloatFilter {
  equals?: number | null;
  not?: number | FloatFilter | null;
  in?: number[];
  notIn?: number[];
  lt?: number;
  lte?: number;
  gt?: number;
  gte?: number;
}

/** Boolean filter operators. */
export interface BoolFilter {
  equals?: boolean | null;
  not?: boolean | BoolFilter | null;
}

/** DateTime filter operators. */
export interface DateTimeFilter {
  equals?: Date | string | null;
  not?: Date | string | DateTimeFilter | null;
  in?: (Date | string)[];
  notIn?: (Date | string)[];
  lt?: Date | string;
  lte?: Date | string;
  gt?: Date | string;
  gte?: Date | string;
}

// ============================================================================
// Query argument types
// ============================================================================

/** Sort order for orderBy. */
export type SortOrder = "asc" | "desc";

/** Base query arguments shared by all find operations. */
export interface FindManyArgs<
  TWhere = any,
  TSelect = any,
  TInclude = any,
  TOrderBy = any,
> {
  where?: TWhere;
  select?: TSelect;
  include?: TInclude;
  orderBy?: TOrderBy | TOrderBy[];
  skip?: number;
  take?: number;
  cursor?: Record<string, any>;
  distinct?: string[];
}

/** Arguments for findUnique. */
export interface FindUniqueArgs<TWhere = any, TSelect = any, TInclude = any> {
  where: TWhere;
  select?: TSelect;
  include?: TInclude;
}

/** Arguments for create. */
export interface CreateArgs<TData = any, TSelect = any, TInclude = any> {
  data: TData;
  select?: TSelect;
  include?: TInclude;
}

/** Arguments for createMany. */
export interface CreateManyArgs<TData = any> {
  data: TData[];
  skipDuplicates?: boolean;
}

/** Arguments for update. */
export interface UpdateArgs<
  TData = any,
  TWhere = any,
  TSelect = any,
  TInclude = any,
> {
  data: TData;
  where: TWhere;
  select?: TSelect;
  include?: TInclude;
}

/** Arguments for updateMany. */
export interface UpdateManyArgs<TData = any, TWhere = any> {
  data: TData;
  where?: TWhere;
}

/** Arguments for delete. */
export interface DeleteArgs<TWhere = any, TSelect = any, TInclude = any> {
  where: TWhere;
  select?: TSelect;
  include?: TInclude;
}

/** Arguments for deleteMany. */
export interface DeleteManyArgs<TWhere = any> {
  where?: TWhere;
}

/** Arguments for upsert. */
export interface UpsertArgs<
  TCreate = any,
  TUpdate = any,
  TWhere = any,
  TSelect = any,
  TInclude = any,
> {
  where: TWhere;
  create: TCreate;
  update: TUpdate;
  select?: TSelect;
  include?: TInclude;
}

/** Arguments for count. */
export interface CountArgs<TWhere = any> {
  where?: TWhere;
  cursor?: Record<string, any>;
  skip?: number;
  take?: number;
  orderBy?: Record<string, SortOrder>;
}

/** Arguments for aggregate. */
export interface AggregateArgs<TWhere = any> {
  where?: TWhere;
  _count?: boolean | Record<string, boolean>;
  _avg?: Record<string, boolean>;
  _sum?: Record<string, boolean>;
  _min?: Record<string, boolean>;
  _max?: Record<string, boolean>;
}

/** Arguments for groupBy. */
export interface GroupByArgs<TWhere = any, THaving = any> {
  by: string[];
  where?: TWhere;
  having?: THaving;
  orderBy?: Record<string, SortOrder> | Record<string, SortOrder>[];
  skip?: number;
  take?: number;
  _count?: boolean | Record<string, boolean>;
  _avg?: Record<string, boolean>;
  _sum?: Record<string, boolean>;
  _min?: Record<string, boolean>;
  _max?: Record<string, boolean>;
}

// ============================================================================
// JSON-RPC protocol types
// ============================================================================

/** JSON-RPC request payload sent to the Go engine. */
export interface RPCRequest {
  jsonrpc: "2.0";
  id: number;
  method: string;
  params: Record<string, unknown>;
}

/** JSON-RPC response payload from the Go engine. */
export interface RPCResponse {
  jsonrpc: "2.0";
  id: number;
  result?: unknown;
  error?: RPCError;
}

/** JSON-RPC error object. */
export interface RPCError {
  code: number;
  message: string;
  data?: unknown;
}

// ============================================================================
// Model delegate interface
// ============================================================================

/** Interface for a model delegate (e.g., client.user). */
export interface ModelDelegate<
  TModel = any,
  TWhere = any,
  TCreate = any,
  TUpdate = any,
  TSelect = any,
  TInclude = any,
  TOrderBy = any,
> {
  findMany(
    args?: FindManyArgs<TWhere, TSelect, TInclude, TOrderBy>,
  ): Promise<TModel[]>;
  findUnique(
    args: FindUniqueArgs<TWhere, TSelect, TInclude>,
  ): Promise<TModel | null>;
  findFirst(
    args?: FindManyArgs<TWhere, TSelect, TInclude, TOrderBy>,
  ): Promise<TModel | null>;
  findUniqueOrThrow(
    args: FindUniqueArgs<TWhere, TSelect, TInclude>,
  ): Promise<TModel>;
  findFirstOrThrow(
    args?: FindManyArgs<TWhere, TSelect, TInclude, TOrderBy>,
  ): Promise<TModel>;
  create(args: CreateArgs<TCreate, TSelect, TInclude>): Promise<TModel>;
  createMany(args: CreateManyArgs<TCreate>): Promise<{ count: number }>;
  update(args: UpdateArgs<TUpdate, TWhere, TSelect, TInclude>): Promise<TModel>;
  updateMany(args: UpdateManyArgs<TUpdate, TWhere>): Promise<{ count: number }>;
  delete(args: DeleteArgs<TWhere, TSelect, TInclude>): Promise<TModel>;
  deleteMany(args?: DeleteManyArgs<TWhere>): Promise<{ count: number }>;
  upsert(
    args: UpsertArgs<TCreate, TUpdate, TWhere, TSelect, TInclude>,
  ): Promise<TModel>;
  count(args?: CountArgs<TWhere>): Promise<number>;
  aggregate(args: AggregateArgs<TWhere>): Promise<Record<string, any>>;
  groupBy(args: GroupByArgs<TWhere>): Promise<Record<string, any>[]>;
  paginate(
    args?: PaginationArgs<TWhere, TSelect, TInclude, TOrderBy>,
  ): Promise<PaginationResult<TModel>>;
  cursorPaginate(
    args?: CursorPaginationArgs<TWhere, TSelect, TInclude, TOrderBy>,
  ): Promise<CursorPaginationResult<TModel>>;
}

// ============================================================================
// Batch payload (for $transaction)
// ============================================================================

/** Batch transaction payload. */
export interface TransactionOptions {
  isolationLevel?:
    | "ReadUncommitted"
    | "ReadCommitted"
    | "RepeatableRead"
    | "Serializable";
  timeout?: number;
}

/** @deprecated Use TransactionOptions instead. */
export type TransactionPayload = TransactionOptions;

/** Practor client options. */
export interface PractorClientOptions {
  /** Path to the Practor engine binary. */
  enginePath?: string;

  /** Path to the schema file. */
  schemaPath?: string;

  /** Database URL (overrides env). */
  datasourceUrl?: string;

  /** Connection pool configuration. */
  pool?: PoolConfig;

  /** Enable query logging. */
  log?: ("query" | "info" | "warn" | "error")[];

  /** Error formatting. */
  errorFormat?: "pretty" | "minimal" | "colorless";
}

// ============================================================================
// Connection pool types
// ============================================================================

/** Configurable connection pool parameters passed to the Go engine. */
export interface PoolConfig {
  /** Maximum number of open connections to the database. Default: 20 */
  maxOpenConns?: number;

  /** Maximum number of idle connections in the pool. Default: 5 */
  maxIdleConns?: number;

  /** Maximum connection lifetime in milliseconds. Default: 300000 (5 min) */
  connMaxLifetimeMs?: number;

  /** Maximum idle time per connection in milliseconds. Default: 60000 (1 min) */
  connMaxIdleTimeMs?: number;
}

/** Runtime connection pool statistics returned by the Go engine. */
export interface PoolStats {
  maxOpenConnections: number;
  openConnections: number;
  inUse: number;
  idle: number;
  waitCount: number;
  waitDurationMs: number;
  maxIdleClosed: number;
  maxIdleTimeClosed: number;
  maxLifetimeClosed: number;
}

// ============================================================================
// Pagination types
// ============================================================================

/** Pagination query arguments. */
export interface PaginationArgs<
  TWhere = any,
  TSelect = any,
  TInclude = any,
  TOrderBy = any,
> {
  where?: TWhere;
  select?: TSelect;
  include?: TInclude;
  orderBy?: TOrderBy | TOrderBy[];
  page?: number;
  limit?: number;
}

/** Pagination result envelope. */
export interface PaginationResult<T = any> {
  data: T[];
  page: number;
  limit: number;
  has_next: boolean;
  total: number;
}

// ============================================================================
// Cursor-based pagination types
// ============================================================================

/** Cursor-based pagination query arguments. */
export interface CursorPaginationArgs<
  TWhere = any,
  TSelect = any,
  TInclude = any,
  TOrderBy = any,
> {
  /** Cursor object identifying the anchor record (e.g. `{ id: 42 }`). Omit for first page. */
  cursor?: Record<string, any>;
  /** Number of records to return per page (default: 10). */
  take?: number;
  where?: TWhere;
  select?: TSelect;
  include?: TInclude;
  /** Required — determines cursor scan direction. */
  orderBy?: TOrderBy | TOrderBy[];
}

/** Cursor-based pagination result envelope. */
export interface CursorPaginationResult<T = any> {
  /** Records for this page. */
  data: T[];
  /** Cursor value for the next page. `null` when on the last page. */
  nextCursor: unknown | null;
  /** Whether more records exist beyond this page. */
  hasNextPage: boolean;
}
