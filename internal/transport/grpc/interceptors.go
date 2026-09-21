package grpcsrv

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryLoggingInterceptor логирует каждый unary-вызов: метод,
// длительность и код ответа.
func UnaryLoggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		log.Info("grpc call",
			"method", info.FullMethod,
			"duration_ms", time.Since(start).Milliseconds(),
			"code", code.String(),
		)

		return resp, err
	}
}

// UnaryRecoveryInterceptor ловит панику в обработчике и превращает
// её в codes.Internal, чтобы сервер не падал.
func UnaryRecoveryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("grpc handler panic",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal error")
			}
		}()

		return handler(ctx, req)
	}
}
