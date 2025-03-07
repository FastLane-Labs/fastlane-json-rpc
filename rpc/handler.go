package rpc

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/FastLane-Labs/fastlane-json-rpc/log"
	"github.com/FastLane-Labs/fastlane-json-rpc/rpc/jsonrpc"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	runtimeMethod      = "RuntimeMethod"
	optionalTypePrefix = "optional_"
)

func (s *Server) handleJsonRpcRequest(ctx context.Context, request *jsonrpc.JsonRpcRequest) *jsonrpc.JsonRpcResponse {
	var (
		start    = time.Now()
		response = s._handleJsonRpcRequest(ctx, request)
		duration = time.Since(start)
	)

	if response.IsSuccess() {
		if s.metrics.enabled {
			s.metrics.RequestDuration.WithLabelValues(request.Method).Observe(duration.Seconds())
		}

		log.Info(ctx, fmt.Sprintf("served %s", request.Method), "duration", duration)
	} else {
		if s.metrics.enabled {
			s.metrics.RequestErrors.Inc()
		}

		log.Warn(ctx, fmt.Sprintf("served %s", request.Method), "duration", duration, "error", response.Error.Error())
	}

	if s.metrics.enabled {
		s.metrics.MethodCalls.WithLabelValues(request.Method).Inc()
	}

	return response
}

func (s *Server) _handleJsonRpcRequest(ctx context.Context, request *jsonrpc.JsonRpcRequest) *jsonrpc.JsonRpcResponse {
	if err := request.Validate(); err != nil {
		return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InvalidRequest, "invalid request", err.Error(), request.Id)
	}

	// Check if method name is reserved
	if request.Method == runtimeMethod {
		return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.MethodNotFound, "method not found: runtimeMethod is reserved", nil, request.Id)
	}

	call := reflect.ValueOf(s.api).MethodByName(cases.Title(language.Und, cases.NoLower).String(request.Method))
	if !call.IsValid() {
		call = s.api.RuntimeMethod(request.Method)
		if !call.IsValid() {
			return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.MethodNotFound, "method not found", nil, request.Id)
		}
	}

	numIn := call.Type().NumIn()
	numParams := len(request.Params)
	hasOptional := hasOptionalInput(numIn, &call)

	// Check if first parameter is context.Context
	hasContextParam := numIn > 0 && call.Type().In(0).String() == "context.Context"
	if hasContextParam {
		numIn-- // Adjust numIn since context will be handled separately
	}

	if !hasValidParamLength(numParams, numIn, hasOptional) {
		return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InvalidParams, "invalid params count", nil, request.Id)
	}

	if isOptionalParamUndefined(numParams, numIn, hasOptional) {
		request.Params = append(request.Params, map[string]interface{}{})
		numParams++
	}

	// Create args slice with room for context if needed
	args := make([]reflect.Value, numParams)
	paramStartIdx := 0

	if hasContextParam {
		// Insert context as first argument and shift other args
		args = make([]reflect.Value, numParams+1)
		args[paramStartIdx] = reflect.ValueOf(ctx)
		paramStartIdx++
	}

	for i, arg := range request.Params {
		paramIndex := i + paramStartIdx
		paramType := call.Type().In(paramIndex)

		// Handle numeric types
		if isNumericType(paramType.Kind()) {
			if val, ok := arg.(float64); ok {
				args[paramIndex] = reflect.ValueOf(convertNumber(val, paramType.Kind()))
				continue
			}
			return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InvalidParams, "invalid params", formatConversionErrMsg(paramIndex, &call), request.Id)
		}

		// Handle other basic types
		switch paramType.Kind() {
		case reflect.Interface:
			args[paramIndex] = reflect.ValueOf(arg)
		case reflect.Map:
			val, ok := arg.(map[string]any)
			if !ok {
				return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InvalidParams, "invalid params", formatConversionErrMsg(paramIndex, &call), request.Id)
			}
			args[paramIndex] = reflect.ValueOf(val)
		case reflect.Slice:
			val, ok := arg.([]interface{})
			if !ok {
				return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InvalidParams, "invalid params", formatConversionErrMsg(paramIndex, &call), request.Id)
			}
			args[paramIndex] = reflect.ValueOf(val)
		case reflect.String:
			val, ok := arg.(string)
			if !ok {
				return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InvalidParams, "invalid params", formatConversionErrMsg(paramIndex, &call), request.Id)
			}
			args[paramIndex] = reflect.ValueOf(val)
		case reflect.Bool:
			val, ok := arg.(bool)
			if !ok {
				return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InvalidParams, "invalid params", formatConversionErrMsg(paramIndex, &call), request.Id)
			}
			args[paramIndex] = reflect.ValueOf(val)
		default:
			return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InternalError, "internal error", "Invalid method definition", request.Id)
		}
	}

	value := call.Call(args)

	if len(value) == 0 {
		// Nothing returned, empty success response
		return jsonrpc.NewJsonRpcSuccessResponse("", request.Id)
	}

	if err, ok := value[len(value)-1].Interface().(error); ok && err != nil {
		// Errored
		return jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InvalidRequest, err.Error(), nil, request.Id)
	}

	if len(value) > 2 {
		// Success with multiple results, return as an array
		result := make([]interface{}, len(value)-1)
		for i := 0; i < len(value)-1; i++ {
			result[i] = value[i].Interface()
		}
		return jsonrpc.NewJsonRpcSuccessResponse(result, request.Id)
	}

	if len(value) == 1 {
		// Success without result, return empty success response
		return jsonrpc.NewJsonRpcSuccessResponse("", request.Id)
	}

	// Success with single result, return as a single value
	return jsonrpc.NewJsonRpcSuccessResponse(value[0].Interface(), request.Id)
}

// hasOptionalInput checks if the API method has defined an optional final input:
//  1. The input must start with the "optional_" prefix in its name.
//  2. The input must be of kind Map.
func hasOptionalInput(numIn int, call *reflect.Value) bool {
	return numIn > 0 &&
		strings.HasPrefix(call.Type().In(numIn-1).Name(), optionalTypePrefix) &&
		call.Type().In(numIn-1).Kind() == reflect.Map
}

// hasValidParamLength checks if the number of parameters in the request is correct:
//  1. Ok if the number of params equals number of method inputs.
//  2. Ok if optional input is defined and number of params is one less the number of method inputs.
func hasValidParamLength(numParams, numIn int, hasOptional bool) bool {
	return numParams == numIn || (hasOptional && numParams == numIn-1)
}

// isOptionalParamUndefined checks if the optional input has been left unset in the request.
func isOptionalParamUndefined(numParams, numIn int, hasOptional bool) bool {
	return hasOptional && numParams == numIn-1
}

func isNumericType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func convertNumber(val float64, kind reflect.Kind) interface{} {
	switch kind {
	case reflect.Int:
		return int(val)
	case reflect.Int8:
		return int8(val)
	case reflect.Int16:
		return int16(val)
	case reflect.Int32:
		return int32(val)
	case reflect.Int64:
		return int64(val)
	case reflect.Uint:
		return uint(val)
	case reflect.Uint8:
		return uint8(val)
	case reflect.Uint16:
		return uint16(val)
	case reflect.Uint32:
		return uint32(val)
	case reflect.Uint64:
		return uint64(val)
	case reflect.Float32:
		return float32(val)
	case reflect.Float64:
		return val
	default:
		return val
	}
}
