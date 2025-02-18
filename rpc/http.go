package rpc

import (
	"context"
	"encoding/json"
	"net/http"

	rpcContext "github.com/FastLane-Labs/fastlane-json-rpc/rpc/context"
	"github.com/FastLane-Labs/fastlane-json-rpc/rpc/jsonrpc"
	"github.com/google/uuid"
)

type HttpRoute struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

func (s *Server) buildHttpRoutes() []HttpRoute {
	return []HttpRoute{
		{
			"HTTP",
			http.MethodPost,
			"/",
			s.httpHandler,
		},
		{
			"HTTP",
			http.MethodGet,
			"/",
			s.httpHandler,
		},
		{
			"HTTP",
			http.MethodGet,
			s.cfg.HealthcheckEndpoint,
			http.HandlerFunc(s.hcCallback),
		},
	}
}

func (s *Server) httpHandler(w http.ResponseWriter, r *http.Request) {
	s.wg.Add(1)
	defer s.wg.Done()

	traceId := r.Header.Get(string(rpcContext.TraceIdLabel))
	if traceId == "" {
		traceId = uuid.New().String()
	}

	ctx := rpcContext.NewContextWithTraceId(context.Background(), traceId)

	if r.Header.Get("Upgrade") == "websocket" {
		if !s.cfg.Websocket.Enabled {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		s.websocketHandler(ctx, w, r)
		return
	}

	if !s.cfg.HTTP.Enabled {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if s.metrics.enabled {
		s.metrics.RequestHttp.Inc()
	}

	var request jsonrpc.JsonRpcRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonrpc.NewJsonRpcErrorResponse(jsonrpc.ParseError, "invalid request", err.Error(), nil).Marshal())
		return
	}

	response := s.handleJsonRpcRequest(ctx, &request)
	if !response.IsSuccess() {
		w.WriteHeader(http.StatusBadRequest)
	}

	w.Write(response.Marshal())
}
