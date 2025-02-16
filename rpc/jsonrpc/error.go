package jsonrpc

import (
	"errors"
)

const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

var (
	ErrInvalidJsonRpcVersion = errors.New("invalid jsonrpc version")
	ErrInvalidJsonRpcId      = errors.New("invalid jsonrpc id")
)
