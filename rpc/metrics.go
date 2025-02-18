package rpc

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	histogramBuckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.45, 0.5, 0.55, 0.6, 0.65, 0.7, 0.8, 0.9, 1.0, 1.5, 2.0, 3.0}
)

type RpcMetrics struct {
	enabled bool

	RequestHttp          prometheus.Counter
	RequestWebsocket     prometheus.Counter
	RequestErrors        prometheus.Counter
	WebsocketConnections prometheus.Gauge
	MethodCalls          *prometheus.CounterVec

	RequestDuration *prometheus.HistogramVec
}

func NewRpcMetrics(reg prometheus.Registerer) *RpcMetrics {
	enabled := reg != nil

	m := &RpcMetrics{enabled: enabled}

	if !enabled {
		return m
	}

	m.RequestHttp = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rpc_request_http",
		Help: "Number of requests served via HTTP",
	})

	m.RequestWebsocket = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rpc_request_websocket",
		Help: "Number of requests served via Websocket",
	})

	m.RequestErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rpc_request_errors",
		Help: "Number of failed requests served",
	})

	m.WebsocketConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rpc_websocket_connections",
		Help: "Number of active websocket connections",
	})

	m.MethodCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_method_calls",
		Help: "Number of method calls",
	}, []string{"method"})

	m.RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rpc_request_duration",
		Help:    "Duration of requests",
		Buckets: histogramBuckets,
	}, []string{"method"})

	reg.MustRegister(
		m.RequestHttp,
		m.RequestWebsocket,
		m.RequestErrors,
		m.WebsocketConnections,
		m.MethodCalls,
		m.RequestDuration,
	)

	return m
}
