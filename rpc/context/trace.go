package context

import (
	_context "context"
)

const (
	TraceIdLabel = TraceIdContextKey("TraceId")
)

type TraceIdContextKey string

func NewContextWithTraceId(ctx _context.Context, traceId string) _context.Context {
	return _context.WithValue(ctx, TraceIdLabel, traceId)
}
