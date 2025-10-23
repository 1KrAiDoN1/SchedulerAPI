package main

import (
	"context"
	"os"
	"scheduler/internal/config"
	"scheduler/internal/handlers"
	httpserver "scheduler/internal/http-server"
	"scheduler/internal/repositories"
	"scheduler/internal/services"
	"scheduler/pkg/database/postgres"
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

	db, err := postgres.NewDatabase(ctx, config.DbConfig.DBConn)
	log.Info(config.DbConfig.DBConn)

	dbpool := db.GetPool()
	defer dbpool.Close()

	repositories := repositories.NewRepositories(dbpool)

	services := services.NewServices(log, repositories)

	handlers := handlers.NewHandlers(log, services)

	server := httpserver.NewServer(log, handlers, config)
	if err := server.Run(); err != nil {
		log.Error("Failed to run HTTP server", slogger.Err(err))
		os.Exit(1)
	}
}
