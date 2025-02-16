package rpc

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/FastLane-Labs/fastlane-json-rpc/log"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

type Server struct {
	cfg     *RpcConfig
	metrics *RpcMetrics
	api     Api

	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

func NewServer(cfg *RpcConfig, enabledMetrics bool, api Api) (*Server, error) {
	s := &Server{
		cfg:          cfg,
		metrics:      NewRpcMetrics(prometheus.DefaultRegisterer, enabledMetrics),
		api:          api,
		shutdownChan: make(chan struct{}),
	}

	if err := s.start(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) start() error {
	return startRpcServer(s.cfg.Port, s.buildHttpRoutes())
}

func (s *Server) Close() {
	close(s.shutdownChan)
	s.wg.Wait()
	log.Info("RPC server stopped")
}

func startRpcServer(port uint64, routes []HttpRoute) error {
	router := mux.NewRouter().StrictSlash(true)
	logger := func(inner func(http.ResponseWriter, *http.Request) error, name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			w.Header().Set("Access-Control-Allow-Origin", "*")
			err := inner(w, r)
			log.Debug(fmt.Sprintf("served %s", name), "method", r.Method, "url", r.RequestURI, "duration", time.Since(start), "error", err)
		})
	}

	for _, route := range routes {
		var handler http.Handler = logger(route.HandlerFunc, route.Name)
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
