package main

import (
	"log/slog"
	"os"

	"github.com/nestera-sh/template/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {

		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	slog.Info("config loaded",
		"http_addr", cfg.HTTPAddr,
		"log_level", cfg.LogLevel,
	)
}
