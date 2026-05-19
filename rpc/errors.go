package rpc

// DataError is an optional interface that errors returned from RPC method
// handlers can implement to attach structured data to the JSON-RPC error
// response's `data` field (per JSON-RPC 2.0). When the handler returns an
// error that implements DataError, the value from ErrorData() is placed in
// the response's `error.data` field. Errors that do not implement this
// interface are unaffected.
type DataError interface {
	error
	ErrorData() interface{}
}
