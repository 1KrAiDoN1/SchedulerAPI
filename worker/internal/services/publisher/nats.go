package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"scheduler/internal/domain/entity"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type ExecutionPublisherInterface interface {
	Publish(ctx context.Context, execution *entity.Execution) error
	Close() error
}

type NATSExecutionPublisher struct {
	js     jetstream.JetStream
	nc     *nats.Conn
	stream jetstream.Stream
	log    *zap.Logger
}

func NewNATSExecutionPublisher(ctx context.Context, log *zap.Logger, natsURL string) (*NATSExecutionPublisher, error) {
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

	log.Info("NATS JetStream publisher initialized successfully")

	return &NATSExecutionPublisher{
		js:     js,
		nc:     nc,
		stream: stream,
		log:    log,
	}, nil
}

func (p *NATSExecutionPublisher) Publish(ctx context.Context, execution *entity.Execution) error {
	// Serialize execution to JSON
	data, err := json.Marshal(execution)
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}

	// Публикуем результат выполнения на subject "JOBS.results"
	subject := "JOBS.results"

	_, err = p.js.Publish(ctx, subject, data)
	if err != nil {
		p.log.Error("Failed to publish execution",
			zap.String("execution_id", execution.ID),
			zap.String("job_id", execution.JobID),
			zap.String("subject", subject),
			zap.Error(err))
		return fmt.Errorf("failed to publish execution to NATS: %w", err)
	}

	p.log.Info("Published execution to NATS",
		zap.String("execution_id", execution.ID),
		zap.String("job_id", execution.JobID),
		zap.String("subject", subject),
		zap.String("status", execution.Status))

	return nil
}

func (p *NATSExecutionPublisher) Close() error {
	if p.nc != nil && !p.nc.IsClosed() {
		p.nc.Close()
	}
	return nil
}
