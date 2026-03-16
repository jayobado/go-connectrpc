package interceptors

import (
	"context"
	"log/slog"
	"fmt"

	"connectrpc.com/connect"
)


type recoveryInterceptor struct{}

func Recovery() connect.Interceptor {
	return &recoveryInterceptor{}
}

func (i *recoveryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (res connect.AnyResponse, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("unary panic recovered",
					slog.String("procedure", req.Spec().Procedure),
					slog.Any("panic", r),
				)
				err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
			}
		}()
		return next(ctx, req)
	}
}

func (i *recoveryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *recoveryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) (err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("streaming panic recovered",
					slog.String("procedure", conn.Spec().Procedure),
					slog.Any("panic", r),
				)
				err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
			}
		}()
		return next(ctx, conn)
	}
}