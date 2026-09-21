package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbv1 "github.com/boba180799/anti-bruteforce/api/proto/v1"
)

// envServer — переменная окружения для адреса сервера.
const envServer = "ABF_SERVER"

// defaultServer — адрес по умолчанию.
const defaultServer = "localhost:9090"

// newRootCmd собирает корневую команду и все подкоманды.
func newRootCmd() *cobra.Command {
	var (
		server  string
		timeout time.Duration
	)

	root := &cobra.Command{
		Use:   "abf-cli",
		Short: "CLI для администрирования сервиса анти-брутфорс",
		Long: "abf-cli управляет whitelist/blacklist и сбрасывает вёдра " +
			"через gRPC-API сервиса анти-брутфорс.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(
		&server, "server", serverFromEnv(), "адрес gRPC-сервера (host:port)",
	)
	root.PersistentFlags().DurationVar(
		&timeout, "timeout", 5*time.Second, "таймаут запроса",
	)

	root.AddCommand(
		newBucketCmd(&server, &timeout),
		newWhitelistCmd(&server, &timeout),
		newBlacklistCmd(&server, &timeout),
		newCheckCmd(&server, &timeout),
	)

	return root
}

// serverFromEnv возвращает адрес сервера из переменной окружения
// или значение по умолчанию.
func serverFromEnv() string {
	if v := os.Getenv(envServer); v != "" {
		return v
	}

	return defaultServer
}

// dialClient создаёт gRPC-клиента и закрывает соединение после
// выполнения команды. Возвращает готовый клиент API.
func dialClient(server string) (pbv1.AntiBruteforceServiceClient, func(), error) {
	conn, err := grpc.NewClient(
		server,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s: %w", server, err)
	}

	client := pbv1.NewAntiBruteforceServiceClient(conn)
	cleanup := func() {
		if cerr := conn.Close(); cerr != nil {
			_ = cerr
		}
	}

	return client, cleanup, nil
}

// withTimeout оборачивает контекст заданным таймаутом.
func withTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}
