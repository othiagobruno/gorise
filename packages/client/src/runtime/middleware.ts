/**
 * @practor/client — Middleware Engine
 *
 * Implements the Prisma-compatible `$use` middleware pattern.
 * Middleware functions form an "onion" call stack — each calls `next(params)`
 * to pass control inward. The innermost handler is the actual engine request.
 *
 * Why onion composition? It allows each middleware to:
 * 1. Inspect/mutate `params` before the query runs (pre-hook)
 * 2. Inspect/mutate the result after the query resolves (post-hook)
 * 3. Short-circuit by returning early without calling `next`
 */

// ============================================================================
// Middleware types
// ============================================================================

/**
 * Parameters passed to each middleware function.
 *
 * Middleware can mutate these before calling `next(params)` to
 * alter the operation that reaches the engine (e.g., rewrite
 * `delete` → `update` for soft-deletes).
 */
export interface MiddlewareParams {
  /** Model name in PascalCase, e.g. "User", "Post" */
  model: string;

  /** CRUD action name, e.g. "findMany", "create", "delete" */
  action: string;

  /** Query arguments (where, data, orderBy, etc.) */
  args: Record<string, unknown>;

  /** RPC method type — "query" for reads, "mutation" for writes */
  method: "query" | "mutation";
}

/**
 * Callback to invoke the next middleware in the chain (or the final handler).
 *
 * Middleware MUST call this to continue the chain, or it can short-circuit
 * by returning a value without calling `next`.
 */
export type MiddlewareNext = (params: MiddlewareParams) => Promise<unknown>;

/**
 * A middleware function registered via `$use()`.
 *
 * @example
 * ```ts
 * const logger: MiddlewareFunction = async (params, next) => {
 *   console.log(`${params.model}.${params.action}`);
 *   return next(params);
 * };
 * practor.$use(logger);
 * ```
 */
export type MiddlewareFunction = (
  params: MiddlewareParams,
  next: MiddlewareNext,
) => Promise<unknown>;

// ============================================================================
// Middleware Engine
// ============================================================================

/**
 * Orchestrates the middleware call stack using FIFO ordering.
 *
 * First registered middleware = outermost wrapper.
 * Last registered middleware = closest to the actual engine call.
 *
 * Composition example with middlewares [A, B, C]:
 * ```
 * A.before → B.before → C.before → engine → C.after → B.after → A.after
 * ```
 */
export class MiddlewareEngine {
  /** Ordered list of registered middleware functions. */
  private middlewares: MiddlewareFunction[] = [];

  /**
   * Registers a middleware function.
   *
   * @param fn - The middleware function to add to the stack
   */
  use(fn: MiddlewareFunction): void {
    if (typeof fn !== "function") {
      throw new TypeError(
        "Middleware must be a function with signature (params, next) => Promise<unknown>",
      );
    }
    this.middlewares.push(fn);
  }

  /**
   * Returns the current number of registered middlewares.
   * Useful for testing and debugging.
   */
  get count(): number {
    return this.middlewares.length;
  }

  /**
   * Executes the full middleware chain, ending with the `finalHandler`.
   *
   * Why build the chain from the tail? We compose the `next` callbacks
   * from inside out: the finalHandler is wrapped first, then each
   * middleware wraps the previous `next`. This ensures FIFO execution
   * order while maintaining the onion model.
   *
   * @param params - The middleware params describing the operation
   * @param finalHandler - The innermost handler (actual engine call)
   * @returns The result from the chain (possibly transformed by middleware)
   */
  async execute(
    params: MiddlewareParams,
    finalHandler: MiddlewareNext,
  ): Promise<unknown> {
    // Fast path: no middleware registered — skip chain construction
    if (this.middlewares.length === 0) {
      return finalHandler(params);
    }

    // Build the chain from the tail (last middleware → finalHandler)
    let next: MiddlewareNext = finalHandler;

    for (let i = this.middlewares.length - 1; i >= 0; i--) {
      const middleware = this.middlewares[i];
      const currentNext = next;

      next = (p: MiddlewareParams) => middleware(p, currentNext);
    }

    return next(params);
  }
}
