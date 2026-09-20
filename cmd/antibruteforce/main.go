package main

import (
	"fmt"
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	logger.Info("Starting Anti-Bruteforce service...", slog.String("version", "1.0.0"))
	fmt.Println("Hello, Anti-Bruteforce!")
}
