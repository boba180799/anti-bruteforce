// Command cli — интерфейс командной строки для администрирования
// сервиса анти-брутфорс: управление whitelist/blacklist и сброс вёдер.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
