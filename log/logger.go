package log

import (
	"context"

	rpcContext "github.com/FastLane-Labs/fastlane-json-rpc/rpc/context"
	gethlog "github.com/ethereum/go-ethereum/log"
)

func Debug(ctx context.Context, format string, v ...interface{}) {
	traceId := ctx.Value(rpcContext.TraceIdLabel)
	if traceId != nil {
		v = append(v, string(rpcContext.TraceIdLabel), traceId)
	}

	gethlog.Debug(format, v...)
}

func Info(ctx context.Context, format string, v ...interface{}) {
	traceId := ctx.Value(rpcContext.TraceIdLabel)
	if traceId != nil {
		v = append(v, string(rpcContext.TraceIdLabel), traceId)
	}

	gethlog.Info(format, v...)
}

func Warn(ctx context.Context, format string, v ...interface{}) {
	traceId := ctx.Value(rpcContext.TraceIdLabel)
	if traceId != nil {
		v = append(v, string(rpcContext.TraceIdLabel), traceId)
	}

	gethlog.Warn(format, v...)
}

func Error(ctx context.Context, format string, v ...interface{}) {
	traceId := ctx.Value(rpcContext.TraceIdLabel)
	if traceId != nil {
		v = append(v, string(rpcContext.TraceIdLabel), traceId)
	}

	gethlog.Error(format, v...)
}
