package main

import (
	"context"
	"os"
	"scheduler/internal/app"
	"scheduler/internal/config"
	"scheduler/pkg/lib/slogger"
)

func main() {
	ctx := context.Background()
	log := slogger.SetupLogger()
	log.Info("Scheduler service started")

	config, err := config.LoadServiceConfig("internal/config/config.yaml", "DB_PASSWORD")
	if err != nil {
		log.Error("Failed to load service config", slogger.Err(err))
		os.Exit(1)
	}

	if err := app.Run(ctx, log, config); err != nil {
		log.Error("Service encountered an error", slogger.Err(err))
		os.Exit(1)
	}

}
