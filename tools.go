//go:build tools

// Package tools фиксирует инструменты сборки и зависимости,
// которые будут использованы в проекте, чтобы go mod tidy
// их не удалял до момента первого импорта.
package tools

import (
	_ "google.golang.org/grpc"
	_ "google.golang.org/protobuf/runtime/protoimpl"
)
