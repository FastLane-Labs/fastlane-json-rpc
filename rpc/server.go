package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/FastLane-Labs/fastlane-json-rpc/log"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

type HealthcheckCallback func(w http.ResponseWriter, r *http.Request)

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

type Server struct {
	cfg     *RpcConfig
	metrics *RpcMetrics
	api     Api

	hcCallback  HealthcheckCallback
	middlewares []Middleware

	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

func NewServer(cfg *RpcConfig, api Api, hcCallback HealthcheckCallback, registerer prometheus.Registerer, middlewares ...Middleware) (*Server, error) {
	gethlog.SetDefault(gethlog.NewLogger(gethlog.NewTerminalHandlerWithLevel(os.Stdout, slog.LevelDebug, true)))

	if hcCallback == nil {
		hcCallback = func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
	}

	s := &Server{
		cfg:          cfg,
		metrics:      NewRpcMetrics(registerer),
		api:          api,
		hcCallback:   hcCallback,
		middlewares:  middlewares,
		shutdownChan: make(chan struct{}),
	}

	if err := startRpcServer(s.cfg.Port, s.buildHttpRoutes(), s.middlewares); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) Close() {
	close(s.shutdownChan)
	s.wg.Wait()
	log.Info(context.Background(), "RPC server stopped")
}

func startRpcServer(port uint64, routes []HttpRoute, middlewares []Middleware) error {
	router := mux.NewRouter().StrictSlash(true)
	logger := func(inner func(http.ResponseWriter, *http.Request)) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			inner(w, r)
		})
	}

	for _, route := range routes {
		var handler http.Handler = logger(route.HandlerFunc)
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	// Apply custom middlewares in reverse order so they execute in the order they were provided
	var finalHandler http.Handler = router
	for i := len(middlewares) - 1; i >= 0; i-- {
		finalHandler = middlewares[i](finalHandler)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handlers.CORS(handlers.AllowedHeaders([]string{"Content-Type"}))(finalHandler),
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return err
	}

	go func() {
		log.Info(context.Background(), "RPC server started", "addr", httpServer.Addr)
		err := httpServer.Serve(ln)
		log.Info(context.Background(), "RPC server stopped", "err", err)
	}()

	return nil
}
