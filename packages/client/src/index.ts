/**
 * @practor/client — Public API
 */

export { PractorClient } from "./runtime/client";
export { PractorEngine, PractorError } from "./runtime/engine";
export {
  MiddlewareEngine,
  type MiddlewareParams,
  type MiddlewareNext,
  type MiddlewareFunction,
} from "./runtime/middleware";
export type {
  // Core types
  PractorClientOptions,
  ModelDelegate,
  TransactionPayload,

  // Pool types
  PoolConfig,
  PoolStats,

  // Filter types
  StringFilter,
  IntFilter,
  FloatFilter,
  BoolFilter,
  DateTimeFilter,

  // Query argument types
  SortOrder,
  FindManyArgs,
  FindUniqueArgs,
  CreateArgs,
  CreateManyArgs,
  UpdateArgs,
  UpdateManyArgs,
  DeleteArgs,
  DeleteManyArgs,
  UpsertArgs,
  CountArgs,
  AggregateArgs,
  GroupByArgs,

  // RPC types
  RPCRequest,
  RPCResponse,
  RPCError,

  // Pagination types
  PaginationArgs,
  PaginationResult,
  CursorPaginationArgs,
  CursorPaginationResult,
} from "./runtime/types";
