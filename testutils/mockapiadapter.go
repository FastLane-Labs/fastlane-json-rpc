package testutils

import (
	"context"
	"errors"
	"reflect"

	rpcContext "github.com/FastLane-Labs/fastlane-json-rpc/rpc/context"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type MockRpcAdapter struct{}

func NewMockRpcAdapter() *MockRpcAdapter {
	return &MockRpcAdapter{}
}

func (r *MockRpcAdapter) RuntimeMethod(methodName string) reflect.Value {
	switch methodName {
	case "mock_runtime_method":
		return reflect.ValueOf(r.Mock_runtime_method)
	}
	return reflect.Value{}
}

func (r *MockRpcAdapter) Mock_methodA(param1 uint64, shouldError bool) (string, error) {
	if shouldError {
		return "", errors.New("mock_methodA error")
	}
	return "mock_methodA success", nil
}

func (r *MockRpcAdapter) Mock_methodB(param1 string, shouldError bool) error {
	if shouldError {
		return errors.New("mock_methodB error")
	}
	return nil
}

func (r *MockRpcAdapter) Mock_methodC(param1 interface{}, shouldError bool) (string, uint64, bool, error) {
	if shouldError {
		return "", 0, false, errors.New("mock_methodC error")
	}
	return "mock_methodC success", 15.0, true, nil
}

func (r *MockRpcAdapter) Mock_methodD(param1 float64, shouldError bool) (string, error) {
	if shouldError {
		return "", errors.New("mock_methodD error")
	}
	return (&hexutil.Bytes{1, 1, 1}).String(), nil
}

func (r *MockRpcAdapter) Mock_runtime_method(param1 float64, shouldError bool) (string, error) {
	if shouldError {
		return "", errors.New("mock_runtime_method error")
	}
	return "mock_runtime_method success", nil
}

func (r *MockRpcAdapter) Mock_methodWithContext(ctx context.Context, param1 float64) (bool, error) {
	if ctx.Value(rpcContext.TraceIdLabel) == nil {
		return false, errors.New("traceId is nil")
	}

	return true, nil
}
