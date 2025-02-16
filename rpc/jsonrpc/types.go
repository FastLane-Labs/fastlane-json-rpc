package jsonrpc

import (
	"encoding/json"
	"fmt"
)

const (
	version = "2.0"
)

type JsonRpcRequest struct {
	Version string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      interface{}   `json:"id"`
}

func (r *JsonRpcRequest) Validate() error {
	if r.Version != version {
		return ErrInvalidJsonRpcVersion
	}

	switch r.Id.(type) {
	case string, float64:
	default:
		return ErrInvalidJsonRpcId
	}

	return nil
}

type JsonRpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewJsonRpcError(code int, message string, data interface{}) *JsonRpcError {
	return &JsonRpcError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func (e *JsonRpcError) Error() string {
	return fmt.Sprintf("code: %d, message: %s, data: %v", e.Code, e.Message, e.Data)
}

type JsonRpcResponse struct {
	Version string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JsonRpcError `json:"error,omitempty"`
	Id      interface{}   `json:"id,omitempty"`
}

func NewJsonRpcSuccessResponse(result interface{}, id interface{}) *JsonRpcResponse {
	if result == nil {
		result = ""
	}

	return &JsonRpcResponse{
		Version: version,
		Result:  result,
		Id:      id,
	}
}

func NewJsonRpcErrorResponse(code int, message string, data interface{}, id interface{}) *JsonRpcResponse {
	return &JsonRpcResponse{
		Version: version,
		Error:   NewJsonRpcError(code, message, data),
		Id:      id,
	}
}

func (r *JsonRpcResponse) IsSuccess() bool {
	return r.Error == nil
}

func (r *JsonRpcResponse) Marshal() []byte {
	json, _ := json.Marshal(r)
	return json
}
