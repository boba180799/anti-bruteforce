// Package httpsrv реализует REST-транспорт сервиса анти-брутфорс
// через grpc-gateway. Слой не содержит бизнес-логики: он
// принимает HTTP-запросы и проксирует их в gRPC.
package httpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	pbv1 "github.com/boba180799/anti-bruteforce/api/proto/v1"
)

// New создаёт http.Server с REST-обёрткой над gRPC-сервером.
// grpcAddr — адрес, на котором слушает gRPC (например, "localhost:9090").
func New(
	ctx context.Context,
	addr string,
	grpcAddr string,
	log *slog.Logger,
) (*http.Server, error) {
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   protojson.MarshalOptions{UseProtoNames: true},
			UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
		}),
	)

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial grpc %s: %w", grpcAddr, err)
	}

	if err := pbv1.RegisterAntiBruteforceServiceHandler(ctx, mux, conn); err != nil {
		return nil, fmt.Errorf("register gateway: %w", err)
	}

	root := http.NewServeMux()
	root.Handle("/v1/", loggingMiddleware(log, mux))
	root.HandleFunc("/healthz", healthz)

	return &http.Server{
		Addr:              addr,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

// healthz — простой health-check для load balancer'ов и k8s.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("ok")); err != nil {
		return
	}
}

// loggingMiddleware логирует каждый HTTP-запрос.
func loggingMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		log.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
