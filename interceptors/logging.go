package interceptors

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)


type loggingInterceptor struct{}

func Logging() connect.Interceptor {
	return &loggingInterceptor{}
}

func (i *loggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		res, err := next(ctx, req)
		slog.Info("unary rpc",
			slog.String("procedure", req.Spec().Procedure),
			slog.Duration("duration", time.Since(start)),
			slog.Any("error", err),
		)
		return res, err
	}
}

func (i *loggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *loggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		err := next(ctx, conn)
		slog.Info("streaming rpc",
			slog.String("procedure", conn.Spec().Procedure),
			slog.Duration("duration", time.Since(start)),
			slog.Any("error", err),
		)
		return err
	}
}