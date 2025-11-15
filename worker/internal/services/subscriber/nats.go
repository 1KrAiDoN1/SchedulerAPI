package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"scheduler/internal/models/dto"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type JobSubscriberInterface interface {
	Subscribe(ctx context.Context, handler func(ctx context.Context, job *dto.JobDTO) error) error
	Close() error
}

type NATSJobSubscriber struct {
	js     jetstream.JetStream
	nc     *nats.Conn
	stream jetstream.Stream
	log    *zap.Logger
	sub    jetstream.Consumer
}

func NewNATSJobSubscriber(ctx context.Context, log *zap.Logger, natsURL string, workerID string) (*NATSJobSubscriber, error) {
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

	// Create consumer for jobs from scheduler
	// Воркер подписывается на джобы со статусом "running"
	consumerConfig := jetstream.ConsumerConfig{
		FilterSubjects: []string{"JOBS.interval.running", "JOBS.once.running"},
		Durable:        fmt.Sprintf("worker-consumer-%s", workerID),
		AckPolicy:      jetstream.AckExplicitPolicy,
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, consumerConfig)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	log.Info("NATS JetStream subscriber initialized successfully",
		zap.String("worker_id", workerID),
		zap.Strings("subjects", consumerConfig.FilterSubjects))

	return &NATSJobSubscriber{
		js:     js,
		nc:     nc,
		stream: stream,
		log:    log,
		sub:    consumer,
	}, nil
}

func (s *NATSJobSubscriber) Subscribe(ctx context.Context, handler func(ctx context.Context, job *dto.JobDTO) error) error {
	consCtx, err := s.sub.Consume(func(msg jetstream.Msg) {
		defer msg.Ack()

		var job dto.JobDTO
		if err := json.Unmarshal(msg.Data(), &job); err != nil {
			s.log.Error("Failed to unmarshal job",
				zap.Error(err),
				zap.String("subject", msg.Subject()))
			return
		}

		if err := handler(ctx, &job); err != nil {
			s.log.Error("Failed to handle job",
				zap.Error(err),
				zap.String("job_id", job.ID),
				zap.String("subject", msg.Subject()))
			return
		}

		s.log.Info("Job received and handler called",
			zap.String("job_id", job.ID),
			zap.String("subject", msg.Subject()))
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer context: %w", err)
	}

	s.log.Info("Subscribed to jobs", zap.Strings("subjects", []string{"JOBS.interval.running", "JOBS.once.running"}))

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
