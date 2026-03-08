/**
 * @practor/client — Public API
 */

export { PractorClient } from "./runtime/client";
export { PractorEngine, PractorError } from "./runtime/engine";
export type {
  // Core types
  PractorClientOptions,
  ModelDelegate,
  TransactionPayload,

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
} from "./runtime/types";
