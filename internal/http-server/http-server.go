package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"scheduler/internal/config"
	"scheduler/internal/http-server/handlers"
	"scheduler/internal/http-server/routes"

	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	//"context"
)

type Server struct {
	router   *gin.Engine
	handlers *handlers.Handlers
	logger   *zap.Logger
	config   config.ServiceConfig
}

func NewServer(logger *zap.Logger, handlers *handlers.Handlers, cfg config.ServiceConfig) *Server {
	router := gin.Default()

	return &Server{
		router:   router,
		handlers: handlers,
		logger:   logger,
		config:   cfg,
	}
}

func (s *Server) Run() error {

	s.setupRoutes()

	// Канал для ошибок сервера
	serverErr := make(chan error, 1)
	go func() {
		s.logger.Info("Starting HTTP server", zap.String("address", s.config.Address))
		if err := s.router.Run(s.config.Address); err != nil {
			serverErr <- fmt.Errorf("server error: %w", err)
		}
		close(serverErr)
	}()

	// Ожидаем сигналы завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Блокируем до получения сигнала или ошибки сервера
	select {
	case err := <-serverErr:
		return err
	case sig := <-quit:
		s.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

		// Получаем доступ к внутреннему http.Server
		srv := &http.Server{
			Addr:    s.config.Address,
			Handler: s.router,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Error("Server forced to shutdown", zap.Error(err))
		}
		s.logger.Info("Server gracefully shut down")
		return nil
	}

}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")

	routes.SetupJobsRoutes(api, s.handlers.JobsHandler)

}
