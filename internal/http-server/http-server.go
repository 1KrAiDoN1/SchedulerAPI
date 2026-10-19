package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"scheduler/internal/config"
	"scheduler/internal/domain/handler"
	"scheduler/internal/http-server/routes"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	server   *http.Server
	router   *gin.Engine
	handlers handler.JobsHandlerInterface
	logger   *zap.Logger
	config   config.ServiceConfig
}

func NewServer(logger *zap.Logger, handlers handler.JobsHandlerInterface, cfg config.ServiceConfig) *Server {
	router := gin.New()

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  config.DefaultReadTimeout,
		WriteTimeout: config.DefaultWriteTimeout,
		IdleTimeout:  config.DefaultIdleTimeout,
	}

	return &Server{
		server:   srv,
		router:   router,
		handlers: handlers,
		logger:   logger,
		config:   cfg,
	}
}

func (s *Server) Run() error {
	s.setupRoutes()

	s.logger.Info("Starting HTTP server", zap.String("address", s.config.Address))
	// ListenAndServe блокирует выполнение до ошибки или shutdown
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.logger.Info("Shutting down HTTP server...")
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", zap.Error(err))
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	s.logger.Info("HTTP server gracefully shut down")
	return nil
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")

	routes.SetupJobsRoutes(api, s.handlers)

}
