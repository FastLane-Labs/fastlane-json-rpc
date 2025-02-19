package context

import (
	_context "context"
	"net/http"
)

var (
	TraceIdLabel = TraceIdContextKey(http.CanonicalHeaderKey("traceId"))
)

type TraceIdContextKey string

func NewContextWithTraceId(ctx _context.Context, traceId string) _context.Context {
	return _context.WithValue(ctx, TraceIdLabel, traceId)
}
