package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"scheduler/internal/domain/entity"
	repositories "scheduler/internal/domain/repository"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

var _ repositories.JobSubscriberInterface = (*NATSJobSubscriber)(nil)

type NATSJobSubscriber struct {
	js     jetstream.JetStream
	nc     *nats.Conn
	stream jetstream.Stream
	log    *zap.Logger
	sub    jetstream.Consumer
}

func NewNATSJobSubscriber(ctx context.Context, log *zap.Logger, natsURL string) (*NATSJobSubscriber, error) {
	maxRetries := 10
	retryDelay := 3 * time.Second
	var nc *nats.Conn
	var err error

	// Retry connection with exponential backoff
	for i := 0; i < maxRetries; i++ {
		opts := []nats.Option{
			nats.MaxReconnects(-1),
			nats.ReconnectWait(2 * time.Second),
			nats.Timeout(10 * time.Second),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				if err != nil {
					log.Warn("NATS disconnected", zap.Error(err))
				}
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Info("NATS reconnected", zap.String("url", natsURL))
			}),
		}

		nc, err = nats.Connect(natsURL, opts...)
		if err == nil {
			break
		}

		log.Warn("Failed to connect to NATS, retrying...",
			zap.String("url", natsURL),
			zap.Error(err),
			zap.Int("attempt", i+1),
			zap.Duration("retry_delay", retryDelay))

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay = time.Duration(float64(retryDelay) * 1.5) // Exponential backoff
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS after %d attempts: %w", maxRetries, err)
	}

	// Wait for connection to be established
	if !nc.IsConnected() {
		nc.Close()
		return nil, fmt.Errorf("NATS connection not established")
	}

	log.Info("Successfully connected to NATS", zap.String("url", natsURL))

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Wait for stream to be available
	var stream jetstream.Stream
	for i := 0; i < maxRetries; i++ {
		stream, err = js.Stream(ctx, "JOBS")
		if err == nil {
			break
		}

		log.Warn("Stream not ready, retrying...",
			zap.String("stream", "JOBS"),
			zap.Error(err),
			zap.Int("attempt", i+1))

		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("stream JOBS not available after %d attempts: %w", maxRetries, err)
	}

	// Create consumer for execution results
	// Воркеры будут отправлять результаты на subject "JOBS.results"
	consumerConfig := jetstream.ConsumerConfig{
		FilterSubjects: []string{"JOBS.results"},
		Durable:        "scheduler-consumer",
		AckPolicy:      jetstream.AckExplicitPolicy,
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, consumerConfig)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	log.Info("NATS JetStream subscriber initialized successfully")

	return &NATSJobSubscriber{
		js:     js,
		nc:     nc,
		stream: stream,
		log:    log,
		sub:    consumer,
	}, nil
}

func (s *NATSJobSubscriber) Subscribe(ctx context.Context, handler func(ctx context.Context, execution *entity.Execution) error) error {
	consCtx, err := s.sub.Consume(func(msg jetstream.Msg) {
		// подтверждаем получение
		defer msg.Ack()

		var execution entity.Execution
		if err := json.Unmarshal(msg.Data(), &execution); err != nil {
			s.log.Error("Failed to unmarshal execution",
				zap.Error(err),
				zap.String("subject", msg.Subject()))
			return
		}

		// Обрабатываем execution в отдельной горутине для параллельной обработки
		go func() {
			if err := handler(ctx, &execution); err != nil {
				s.log.Error("Failed to handle execution",
					zap.Error(err),
					zap.String("execution_id", execution.ID),
					zap.String("job_id", execution.JobID))
				return
			}

			s.log.Info("Execution processed successfully",
				zap.String("execution_id", execution.ID),
				zap.String("job_id", execution.JobID),
				zap.String("status", execution.Status))
		}()
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer context: %w", err)
	}

	s.log.Info("Subscribed to execution results", zap.String("subject", "JOBS.results"))

	// Блокируем выполнение до отмены контекста
	<-ctx.Done()
	consCtx.Stop()
	s.log.Info("NATS subscriber stopped")

	return nil
}

func (s *NATSJobSubscriber) Close() error {
	if s.nc != nil && !s.nc.IsClosed() {
		s.nc.Close()
	}
	return nil
}
