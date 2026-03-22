package interceptors

import (
	"context"
	"errors"
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

		attrs := []any{
			slog.String("procedure", req.Spec().Procedure),
			slog.Duration("duration", time.Since(start)),
		}
		if err != nil {
			var connectErr *connect.Error
			if errors.As(err, &connectErr) {
				attrs = append(attrs,
					slog.String("code", connectErr.Code().String()),
					slog.Any("error", err),
				)
			} else {
				attrs = append(attrs, slog.Any("error", err))
			}
		}

		slog.Info("unary rpc", attrs...)
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

		attrs := []any{
			slog.String("procedure", conn.Spec().Procedure),
			slog.Duration("duration", time.Since(start)),
		}
		if err != nil {
			var connectErr *connect.Error
			if errors.As(err, &connectErr) {
				attrs = append(attrs,
					slog.String("code", connectErr.Code().String()),
					slog.Any("error", err),
				)
			} else {
				attrs = append(attrs, slog.Any("error", err))
			}
		}

		slog.Info("streaming rpc", attrs...)
		return err
	}
}