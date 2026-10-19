package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"scheduler/internal/domain/entity"
	"scheduler/internal/models/dto"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type NATSJobPublisher struct {
	js     jetstream.JetStream
	nc     *nats.Conn
	stream jetstream.Stream
	log    *zap.Logger
}

func NewNATSJobPublisher(ctx context.Context, log *zap.Logger, natsURL string) (*NATSJobPublisher, error) {
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

	return &NATSJobPublisher{
		js:     js,
		nc:     nc,
		stream: stream,
		log:    log,
	}, nil
}

func (p *NATSJobPublisher) Close() error {
	if p.nc != nil && !p.nc.IsClosed() {
		p.nc.Close()
	}
	return nil
}

func (p *NATSJobPublisher) Publish(ctx context.Context, job *entity.Job) error {
	// Convert entity to JSON-serializable DTO
	dto := dto.JobDTO{
		ID:             job.ID,
		Kind:           int(job.Kind),
		Status:         string(job.Status),
		LastFinishedAt: job.LastFinishedAt,
		Payload:        job.Payload,
	}

	// Convert interval if present
	if job.Interval != nil {
		durationStr := job.Interval.String()
		dto.Interval = &durationStr
	}

	// Convert once if present
	if job.Once != nil {
		dto.Once = job.Once
	}

	// Serialize job to JSON
	data, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if job.Kind == entity.JobUndefined {
		return errors.New("undefined job kind")
	}

	// Construct subject based on job kind and status
	subject := p.subjectForJob(job)

	// Log before publishing for debugging
	p.log.Debug("Attempting to publish job",
		zap.String("job_id", job.ID),
		zap.String("subject", subject),
		zap.Int("data_size", len(data)))

	// Use synchronous Publish with timeout context
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ack, err := p.js.Publish(publishCtx, subject, data)
	if err != nil {
		p.log.Error("Failed to publish job to NATS",
			zap.String("job_id", job.ID),
			zap.String("subject", subject),
			zap.Error(err))
		return fmt.Errorf("failed to publish job to NATS: %w", err)
	}

	p.log.Info("Published job to NATS",
		zap.String("job_id", job.ID),
		zap.String("subject", subject),
		zap.Uint64("stream_sequence", ack.Sequence))

	return nil
}

func (p *NATSJobPublisher) subjectForJob(job *entity.Job) string {
	switch job.Kind {
	case entity.JobKindInterval:
		return fmt.Sprintf("JOBS.interval.%s", job.Status)
	case entity.JobKindOnce:
		return fmt.Sprintf("JOBS.once.%s", job.Status)
	}

	return ""
}
