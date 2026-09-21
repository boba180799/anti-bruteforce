// Command antibruteforce — точка входа сервиса анти-брутфорс.
// Запускает gRPC-сервер, подключает usecase и хранилища.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	pbv1 "github.com/boba180799/anti-bruteforce/api/proto/v1"
	"github.com/boba180799/anti-bruteforce/internal/config"
	"github.com/boba180799/anti-bruteforce/internal/limiter"
	"github.com/boba180799/anti-bruteforce/internal/repository/memory"
	grpcsrv "github.com/boba180799/anti-bruteforce/internal/transport/grpc"
	"github.com/boba180799/anti-bruteforce/internal/usecase"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(log)

	cfg := config.LoadFromEnv()
	log.Info("starting anti-bruteforce",
		"grpc_addr", cfg.GRPCAddr,
		"http_addr", cfg.HTTPAddr,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	storage := limiter.NewStorage(limiter.StorageConfig{
		LoginCapacity:    cfg.LoginLimit,
		LoginRate:        cfg.LoginRate,
		PasswordCapacity: cfg.PasswordLimit,
		PasswordRate:     cfg.PasswordRate,
		IPCapacity:       cfg.IPLimit,
		IPRate:           cfg.IPRate,
		TTL:              cfg.BucketTTL,
	})
	storage.Start(ctx)

	rules := memory.NewIPRules()

	checkUC := usecase.NewCheckAttempt(storage, rules)
	resetUC := usecase.NewResetBucket(storage)
	rulesUC := usecase.NewManageRules(rules)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcsrv.UnaryRecoveryInterceptor(log),
			grpcsrv.UnaryLoggingInterceptor(log),
		),
	)
	pbv1.RegisterAntiBruteforceServiceServer(grpcServer, grpcsrv.NewServer(checkUC, resetUC, rulesUC, log))

	var lc net.ListenConfig

	lis, err := lc.Listen(ctx, "tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.GRPCAddr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("grpc server started", "addr", cfg.GRPCAddr)
		if serveErr := grpcServer.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err = <-errCh:
		return err
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Info("grpc server stopped gracefully")
	case <-time.After(cfg.ShutdownTimeout):
		log.Warn("graceful stop timeout, forcing")
		grpcServer.Stop()
	}

	return nil
}
