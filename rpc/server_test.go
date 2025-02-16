package rpc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/FastLane-Labs/fastlane-json-rpc/testutils"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

func TestServer_HttpRequest(t *testing.T) {
	testCfg := &RpcConfig{
		Port: 8080,
		HTTP: &HttpConfig{
			Enabled: true,
		},
		Websocket: &WebsocketConfig{},
	}

	api := testutils.NewMockRpcAdapter()

	s, err := NewServer(testCfg, false, api)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	defer s.Close()

	tt := []struct {
		methodCalled      string
		methodParams      []interface{}
		expectedHttpCode  int
		expectedResult    interface{}
		expectedErrorPart string
	}{
		{
			methodCalled:      "invalid_method",
			methodParams:      []interface{}{60, false},
			expectedHttpCode:  http.StatusBadRequest,
			expectedErrorPart: "method not found",
		},
		{
			methodCalled:      "mock_methodA",
			methodParams:      []interface{}{60, false, 500},
			expectedHttpCode:  http.StatusBadRequest,
			expectedErrorPart: "invalid params count",
		},
		{
			methodCalled:      "mock_methodA",
			methodParams:      []interface{}{"60", false},
			expectedHttpCode:  http.StatusBadRequest,
			expectedErrorPart: "invalid params",
		},
		{
			methodCalled:     "mock_methodA",
			methodParams:     []interface{}{60, false},
			expectedHttpCode: http.StatusOK,
			expectedResult:   "mock_methodA success",
		},
		{
			methodCalled:      "mock_methodA",
			methodParams:      []interface{}{60, true},
			expectedHttpCode:  http.StatusBadRequest,
			expectedErrorPart: "mock_methodA error",
		},
		{
			methodCalled:      "mock_runtime_method",
			methodParams:      []interface{}{60, false},
			expectedHttpCode:  http.StatusOK,
			expectedResult:   "mock_runtime_method success",
		},
		{
			methodCalled:      "mock_runtime_method",
			methodParams:      []interface{}{60, true},
			expectedHttpCode:  http.StatusBadRequest,
			expectedErrorPart: "mock_runtime_method error",
		},
		{
			methodCalled:      "mock_runtime_method_unknown",
			methodParams:      []interface{}{60, false},
			expectedHttpCode:  http.StatusBadRequest,
			expectedErrorPart: "method not found",
		},
	}

	for _, tc := range tt {
		rpcReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  tc.methodCalled,
			"params":  tc.methodParams,
			"id":      "1",
		}

		rpcReqBytes, err := json.Marshal(rpcReq)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		resp, err := http.Post("http://localhost:8080", "application/json", bytes.NewBuffer(rpcReqBytes))
		if err != nil {
			t.Fatalf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, tc.expectedHttpCode, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		rpcResp := struct {
			Result interface{} `json:"result"`
			Error  interface{} `json:"error"`
		}{}
		if err := json.Unmarshal(body, &rpcResp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if rpcResp.Result != nil {
			assert.Equal(t, tc.expectedResult, rpcResp.Result)
		}

		if rpcResp.Error != nil {
			assert.Contains(t, rpcResp.Error.(map[string]interface{})["message"], tc.expectedErrorPart)
		}
	}
}

func TestServer_WebsocketRequest(t *testing.T) {
	testCfg := &RpcConfig{
		Port: 8081,
		Websocket: &WebsocketConfig{
			Enabled: true,
		},
		HTTP: &HttpConfig{},
	}

	api := testutils.NewMockRpcAdapter()

	s, err := NewServer(testCfg, false, api)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	defer s.Close()

	rpcClient, err := ethclient.Dial("ws://localhost:8081")
	if err != nil {
		t.Fatalf("failed to create rpc client: %v", err)
	}

	defer rpcClient.Close()

	tt := []struct {
		methodCalled      string
		methodParams      []interface{}
		expectedSuccess   bool
		expectedResult    interface{}
		expectedErrorPart string
	}{
		{
			methodCalled:    "mock_methodB",
			methodParams:    []interface{}{"param", false},
			expectedSuccess: true,
			expectedResult:  "",
		},
		{
			methodCalled:      "mock_methodB",
			methodParams:      []interface{}{"param", true},
			expectedSuccess:   false,
			expectedErrorPart: "mock_methodB error",
		},
		{
			methodCalled:    "mock_methodC",
			methodParams:    []interface{}{[]string{"interface"}, false},
			expectedSuccess: true,
			expectedResult:  []interface{}{"mock_methodC success", 15.0, true},
		},
		{
			methodCalled:      "mock_methodC",
			methodParams:      []interface{}{999, true},
			expectedSuccess:   false,
			expectedErrorPart: "mock_methodC error",
		},
		{
			methodCalled:    "mock_methodD",
			methodParams:    []interface{}{12.34, false},
			expectedSuccess: true,
			expectedResult:  "0x010101",
		},
		{
			methodCalled:      "mock_methodD",
			methodParams:      []interface{}{99.99, true},
			expectedSuccess:   false,
			expectedErrorPart: "mock_methodD error",
		},
	}

	for _, tc := range tt {
		var result interface{}
		if err := rpcClient.Client().Call(
			&result,
			tc.methodCalled,
			tc.methodParams...,
		); err != nil {
			assert.Equal(t, tc.expectedSuccess, false)
			assert.Contains(t, err.Error(), tc.expectedErrorPart)
		} else {
			assert.Equal(t, tc.expectedSuccess, true)
			assert.Equal(t, tc.expectedResult, result)
		}
	}
}
