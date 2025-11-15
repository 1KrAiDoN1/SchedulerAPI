package main

import (
	"context"
	"os"
	"scheduler/worker/internal/app"
	"scheduler/worker/internal/config"
	"scheduler/worker/pkg/lib/logger/zaplogger"

	"go.uber.org/zap/zapcore"
)

func main() {
	ctx := context.Background()
	log := zaplogger.SetupLoggerWithLevel(zapcore.DebugLevel)
	log.Info("Worker service started")

	cfg, err := config.LoadConfig(log, "worker/internal/config/config.yaml")
	if err != nil {
		log.Error("Failed to load config from yaml", zaplogger.Err(err))
	}

	if err := app.Run(ctx, log, cfg); err != nil {
		log.Error("Failed to run worker service", zaplogger.Err(err))
		os.Exit(1)
	}
}
