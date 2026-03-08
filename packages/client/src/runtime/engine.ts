/**
 * @practor/client — Go Engine Process Manager
 *
 * Spawns the Go query engine binary as a child process and manages
 * JSON-RPC communication over stdin/stdout.
 */

import { ChildProcess, spawn } from "child_process";
import { EventEmitter } from "events";
import * as path from "path";
import * as readline from "readline";
import type { RPCRequest, RPCResponse, RPCError } from "./types";

/** Engine connection state. */
type EngineState = "idle" | "starting" | "ready" | "error" | "stopped";

/** Pending request waiting for a response. */
interface PendingRequest {
  resolve: (value: unknown) => void;
  reject: (reason: Error) => void;
  timer: NodeJS.Timeout;
}

/**
 * PractorEngine manages the Go engine child process.
 *
 * Why a child process? This mirrors Prisma's architecture. The Go binary
 * runs as a sidecar, communicating via line-delimited JSON-RPC over
 * stdin/stdout. This provides process isolation, crash safety, and
 * avoids native module compilation.
 */
export class PractorEngine extends EventEmitter {
  private process: ChildProcess | null = null;
  private state: EngineState = "idle";
  private requestId = 0;
  private pending = new Map<number, PendingRequest>();
  private enginePath: string;
  private schemaPath: string;
  private datasourceUrl?: string;
  private requestTimeout: number;

  constructor(
    options: {
      enginePath?: string;
      schemaPath?: string;
      datasourceUrl?: string;
      requestTimeout?: number;
    } = {},
  ) {
    super();
    this.enginePath = options.enginePath || this.resolveEnginePath();
    this.schemaPath = options.schemaPath || "schema.practor";
    this.datasourceUrl = options.datasourceUrl;
    this.requestTimeout = options.requestTimeout || 30_000;
  }

  /** Resolves the path to the Go engine binary. */
  private resolveEnginePath(): string {
    // Check for custom PRACTOR_ENGINE_PATH env var
    const envPath = process.env.PRACTOR_ENGINE_PATH;
    if (envPath) return envPath;

    // Default: look in the project's bin directory
    return path.resolve(
      process.cwd(),
      "node_modules",
      ".practor",
      "practor-engine",
    );
  }

  /** Starts the engine process and waits for the ready signal. */
  async start(): Promise<void> {
    if (this.state === "ready") return;
    if (this.state === "starting") {
      return new Promise((resolve, reject) => {
        this.once("ready", resolve);
        this.once("error", reject);
      });
    }

    this.state = "starting";

    const env: Record<string, string> = {
      ...(process.env as Record<string, string>),
    };
    env.PRACTOR_SCHEMA_PATH = this.schemaPath;
    if (this.datasourceUrl) {
      env.DATABASE_URL = this.datasourceUrl;
    }

    this.process = spawn(this.enginePath, [], {
      stdio: ["pipe", "pipe", "pipe"],
      env,
    });

    // Read stdout line by line for JSON-RPC responses
    const rl = readline.createInterface({
      input: this.process.stdout!,
      crlfDelay: Infinity,
    });

    rl.on("line", (line: string) => {
      this.handleResponse(line);
    });

    // Capture stderr for logging
    this.process.stderr?.on("data", (data: Buffer) => {
      const msg = data.toString().trim();
      if (msg) {
        this.emit("log", msg);
      }
    });

    this.process.on("exit", (code: number | null) => {
      this.state = "stopped";
      this.rejectAllPending(
        new Error(`Engine process exited with code ${code}`),
      );
      this.emit("exit", code);
    });

    this.process.on("error", (err: Error) => {
      this.state = "error";
      this.rejectAllPending(err);
      this.emit("error", err);
    });

    // Wait for the initial ready message
    return new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error("Engine startup timeout (10s)"));
      }, 10_000);

      const handler = (line: string) => {
        try {
          const msg = JSON.parse(line) as RPCResponse;
          if (msg.id === 0 && msg.result) {
            clearTimeout(timeout);
            rl.removeListener("line", handler);
            this.state = "ready";
            this.emit("ready");
            resolve();
          }
        } catch {
          // Ignore non-JSON lines during startup
        }
      };

      // Re-attach handler temporarily to catch the ready message
      rl.removeAllListeners("line");
      rl.on("line", handler);

      // After ready, switch back to normal response handling
      this.once("ready", () => {
        rl.removeAllListeners("line");
        rl.on("line", (line: string) => this.handleResponse(line));
      });
    });
  }

  /** Stops the engine process gracefully. */
  async stop(): Promise<void> {
    if (!this.process || this.state === "stopped") return;

    try {
      await this.request("shutdown", {});
    } catch {
      // Ignore errors during shutdown
    }

    this.process.kill("SIGTERM");
    this.process = null;
    this.state = "stopped";
    this.rejectAllPending(new Error("Engine stopped"));
  }

  /** Sends a JSON-RPC request to the engine and waits for a response. */
  async request(
    method: string,
    params: Record<string, unknown>,
  ): Promise<unknown> {
    if (this.state !== "ready" && method !== "shutdown") {
      throw new Error(`Engine is not ready (state: ${this.state})`);
    }

    const id = ++this.requestId;
    const req: RPCRequest = {
      jsonrpc: "2.0",
      id,
      method,
      params,
    };

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(
          new Error(
            `Request timeout for method '${method}' (${this.requestTimeout}ms)`,
          ),
        );
      }, this.requestTimeout);

      this.pending.set(id, { resolve, reject, timer });

      const json = JSON.stringify(req) + "\n";
      this.process!.stdin!.write(json);
    });
  }

  /** Handles a JSON-RPC response from the engine. */
  private handleResponse(line: string): void {
    let response: RPCResponse;
    try {
      response = JSON.parse(line);
    } catch {
      return; // Ignore non-JSON lines
    }

    const pending = this.pending.get(response.id);
    if (!pending) return;

    this.pending.delete(response.id);
    clearTimeout(pending.timer);

    if (response.error) {
      const err = new PractorError(
        response.error.message,
        response.error.code,
        response.error.data,
      );
      pending.reject(err);
    } else {
      pending.resolve(response.result);
    }
  }

  /** Rejects all pending requests. */
  private rejectAllPending(error: Error): void {
    for (const [id, pending] of this.pending) {
      clearTimeout(pending.timer);
      pending.reject(error);
    }
    this.pending.clear();
  }

  /** Returns the engine state. */
  getState(): EngineState {
    return this.state;
  }
}

/**
 * PractorError represents an error from the query engine.
 */
export class PractorError extends Error {
  readonly code: number;
  readonly data: unknown;

  constructor(message: string, code: number, data?: unknown) {
    super(message);
    this.name = "PractorError";
    this.code = code;
    this.data = data;
  }
}
