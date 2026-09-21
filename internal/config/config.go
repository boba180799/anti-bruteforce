// Package config загружает конфигурацию сервиса из переменных окружения.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config — конфигурация сервиса анти-брутфорс.
type Config struct {
	// GRPCAddr — адрес gRPC-сервера (например, ":9090").
	GRPCAddr string

	// HTTPAddr — адрес REST-сервера (например, ":8080").
	HTTPAddr string

	// LoginLimit — максимум одновременных попыток по логину.
	LoginLimit float64

	// LoginRate — сколько единиц восстанавливается за секунду.
	LoginRate float64

	// PasswordLimit — максимум одновременных попыток по паролю.
	PasswordLimit float64

	// PasswordRate — скорость восстановления для пароля.
	PasswordRate float64

	// IPLimit — максимум одновременных попыток по IP.
	IPLimit float64

	// IPRate — скорость восстановления для IP.
	IPRate float64

	// BucketTTL — время жизни неактивного ведра.
	BucketTTL time.Duration

	// ShutdownTimeout — максимальное время ожидания graceful shutdown.
	ShutdownTimeout time.Duration
}

// LoadFromEnv читает конфигурацию из переменных окружения
// и подставляет значения по умолчанию.
func LoadFromEnv() Config {
	return Config{
		GRPCAddr: envOr("GRPC_ADDR", ":9090"),
		HTTPAddr: envOr("HTTP_ADDR", ":8080"),

		// Лимиты по умолчанию из ТЗ:
		// N=10 (login), M=100 (password), K=1000 (ip) в минуту.
		LoginLimit:    envFloat("LOGIN_LIMIT", 10),
		LoginRate:     envFloat("LOGIN_RATE", 10.0/60),
		PasswordLimit: envFloat("PASSWORD_LIMIT", 100),
		PasswordRate:  envFloat("PASSWORD_RATE", 100.0/60),
		IPLimit:       envFloat("IP_LIMIT", 1000),
		IPRate:        envFloat("IP_RATE", 1000.0/60),

		BucketTTL:       envDuration("BUCKET_TTL", 10*time.Minute),
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}

func envFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}

	return f
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}

	return d
}
