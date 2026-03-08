package protocol

import (
	"encoding/json"
)

// ============================================================================
// JSON-RPC Types — Communication protocol between Node.js client and Go engine
// ============================================================================

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603

	// Custom error codes
	ErrCodeNotFound       = -32001
	ErrCodeValidation     = -32002
	ErrCodeDatabase       = -32003
	ErrCodeSchema         = -32004
)

// ============================================================================
// Query request parameters
// ============================================================================

// QueryParams represents the parameters for a query/mutation request.
type QueryParams struct {
	Model  string                 `json:"model"`
	Action string                 `json:"action"` // findMany, findUnique, create, update, delete, etc.
	Args   map[string]interface{} `json:"args"`
}

// MigrateParams represents the parameters for a migration request.
type MigrateParams struct {
	Action     string `json:"action"`     // dev, deploy, reset, status
	SchemaPath string `json:"schemaPath"`
	Name       string `json:"name,omitempty"`
}

// SchemaParams represents the parameters for schema operations.
type SchemaParams struct {
	Action     string `json:"action"`     // parse, validate, format
	SchemaPath string `json:"schemaPath,omitempty"`
	Schema     string `json:"schema,omitempty"`
}

// IntrospectParams represents parameters for database introspection.
type IntrospectParams struct {
	Schema string `json:"schema,omitempty"` // Existing schema to merge with
}

// DbPushParams represents parameters for db push operations.
type DbPushParams struct {
	SchemaPath    string `json:"schemaPath"`
	AcceptDataLoss bool  `json:"acceptDataLoss"`
	ForceReset    bool   `json:"forceReset"`
}

// TransactionBeginParams represents parameters for starting a transaction.
type TransactionBeginParams struct {
	IsolationLevel string `json:"isolationLevel,omitempty"`
	Timeout        int    `json:"timeout,omitempty"` // milliseconds
}

// TransactionActionParams represents parameters for executing within a transaction.
type TransactionActionParams struct {
	TxID   string                 `json:"txId"`
	Model  string                 `json:"model"`
	Action string                 `json:"action"`
	Args   map[string]interface{} `json:"args"`
}

// TransactionIDParams represents parameters referencing a transaction by ID.
type TransactionIDParams struct {
	TxID string `json:"txId"`
}

// TransactionBeginResponse represents the response from starting a transaction.
type TransactionBeginResponse struct {
	TxID string `json:"txId"`
}

// PaginationResponse represents a paginated query result.
type PaginationResponse struct {
	Data    interface{} `json:"data"`
	Page    int         `json:"page"`
	Limit   int         `json:"limit"`
	HasNext bool        `json:"has_next"`
	Total   int64       `json:"total"`
}

// ============================================================================
// Query result types
// ============================================================================

// QueryResponse represents the result of a query operation.
type QueryResponse struct {
	Data  interface{} `json:"data"`
	Count int64       `json:"count,omitempty"`
}

// MutationResponse represents the result of a mutation operation.
type MutationResponse struct {
	Data    interface{} `json:"data"`
	Count   int64       `json:"count,omitempty"`
}

// SchemaResponse represents the result of a schema operation.
type SchemaResponse struct {
	Schema interface{} `json:"schema,omitempty"`
	Valid  bool        `json:"valid"`
	Errors []string    `json:"errors,omitempty"`
}

// ============================================================================
// Helper constructors
// ============================================================================

// NewSuccessResponse creates a success response.
func NewSuccessResponse(id int, result interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// NewErrorResponse creates an error response.
func NewErrorResponse(id int, code int, message string, data interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// NewNotFoundError creates a "not found" error response.
func NewNotFoundError(id int, model string) *Response {
	return NewErrorResponse(id, ErrCodeNotFound,
		"Record not found",
		map[string]string{"model": model})
}

// NewValidationError creates a validation error response.
func NewValidationError(id int, message string) *Response {
	return NewErrorResponse(id, ErrCodeValidation, message, nil)
}
