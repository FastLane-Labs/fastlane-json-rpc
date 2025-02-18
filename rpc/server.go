package rpc

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/FastLane-Labs/fastlane-json-rpc/log"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

type HealthcheckCallback func(w http.ResponseWriter, r *http.Request)

type Server struct {
	cfg     *RpcConfig
	metrics *RpcMetrics
	api     Api

	hcCallback HealthcheckCallback

	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

func NewServer(cfg *RpcConfig, api Api, hcCallback HealthcheckCallback, registerer prometheus.Registerer) (*Server, error) {
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
		shutdownChan: make(chan struct{}),
	}

	if err := startRpcServer(s.cfg.Port, s.buildHttpRoutes()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) Close() {
	close(s.shutdownChan)
	s.wg.Wait()
	log.Info("RPC server stopped")
}

func startRpcServer(port uint64, routes []HttpRoute) error {
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

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handlers.CORS(handlers.AllowedHeaders([]string{"Content-Type"}))(router),
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return err
	}

	go func() {
		log.Info("RPC server started", "addr", httpServer.Addr)
		err := httpServer.Serve(ln)
		log.Info("RPC server stopped", "err", err)
	}()

	return nil
}
