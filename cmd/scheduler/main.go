package main

import (
	"context"
	"os"
	"scheduler/internal/app"
	"scheduler/internal/config"
	"scheduler/pkg/lib/logger/zaplogger"

	"go.uber.org/zap/zapcore"
)

func main() {
	ctx := context.Background()
	log := zaplogger.SetupLoggerWithLevel(zapcore.DebugLevel)
	log.Info("Scheduler service started")
	config, err := config.LoadServiceConfig(log, "internal/config/config.yaml", "DB_PASSWORD")
	if err != nil {
		log.Error("Failed to load service config", zaplogger.Err(err))
		os.Exit(1)
	}

	if err := app.Run(ctx, log, config); err != nil {
		log.Error("Failed to Run application service", zaplogger.Err(err))
		os.Exit(1)
	}

}
