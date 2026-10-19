package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"scheduler/internal/domain/entity"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrJobNotFound       = errors.New("job not found")
	ErrExecutionNotFound = errors.New("execution not found")
)

type JobsRepository struct {
	pool *pgxpool.Pool
}

func NewJobsRepository(pool *pgxpool.Pool) *JobsRepository {
	return &JobsRepository{
		pool: pool,
	}
}

const (
	queryCreateJob = `INSERT INTO jobs (id, kind, status, interval_seconds, once_timestamp, last_finished_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO NOTHING`

	queryRead = `SELECT id, kind, status, interval_seconds, once_timestamp, last_finished_at, payload FROM jobs WHERE id = $1`

	queryList = `SELECT id, kind, status, interval_seconds, once_timestamp, last_finished_at, payload FROM jobs ORDER BY id`

	queryUpsert = `INSERT INTO jobs (id, kind, status, interval_seconds, once_timestamp, last_finished_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO UPDATE SET 
		status = EXCLUDED.status,interval_seconds = EXCLUDED.interval_seconds, last_finished_at = EXCLUDED.last_finished_at`

	queryDelete = `DELETE FROM jobs WHERE id = $1`
)

func (j *JobsRepository) Create(ctx context.Context, job *entity.Job) error {
	payload, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal job's payload: %w", err)
	}

	var intervalSeconds *int64
	if job.Interval != nil {
		seconds := int64(job.Interval.Seconds())
		intervalSeconds = &seconds
	}

	_, err = j.pool.Exec(ctx, queryCreateJob, job.ID,
		job.Kind,
		job.Status,
		sql.NullInt64{
			Int64: func() int64 {
				if intervalSeconds != nil {
					return *intervalSeconds
				}
				return 0
			}(),
			Valid: intervalSeconds != nil,
		},
		sql.NullInt64{
			Int64: func() int64 {
				if job.Once != nil {
					return *job.Once
				}
				return 0
			}(),
			Valid: job.Once != nil,
		},
		job.LastFinishedAt,
		payload)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	return nil
}

func (j *JobsRepository) Read(ctx context.Context, jobID string) (*entity.Job, error) {
	var (
		id              string
		kind            int
		status          string
		intervalSeconds sql.NullInt64
		onceTimestamp   sql.NullInt64
		lastFinishedAt  int64
		payloadJSON     []byte
	)

	err := j.pool.QueryRow(ctx, queryRead, jobID).Scan(
		&id,
		&kind,
		&status,
		&intervalSeconds,
		&onceTimestamp,
		&lastFinishedAt,
		&payloadJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query row: %w", err)
	}

	var payload any
	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			return nil, fmt.Errorf("unmarshal payload: %w", err)
		}
	} else {
		return nil, fmt.Errorf("nil job's payload")
	}

	var interval *time.Duration
	if intervalSeconds.Valid {
		dur := time.Duration(intervalSeconds.Int64) * time.Second
		interval = &dur
	}

	var once *int64
	if onceTimestamp.Valid {
		once = &onceTimestamp.Int64
	}

	return &entity.Job{
		ID:             id,
		Kind:           entity.JobKind(kind),
		Status:         entity.JobStatus(status),
		Interval:       interval,
		Once:           once,
		LastFinishedAt: lastFinishedAt,
		Payload:        payload,
	}, nil
}

func (j *JobsRepository) List(ctx context.Context) ([]*entity.Job, error) {
	rows, err := j.pool.Query(ctx, queryList)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*entity.Job
	for rows.Next() {
		var (
			id              string
			kind            int
			status          string
			intervalSeconds sql.NullInt64
			onceTimestamp   sql.NullInt64
			lastFinishedAt  int64
			payloadJSON     []byte
		)

		if err := rows.Scan(
			&id,
			&kind,
			&status,
			&intervalSeconds,
			&onceTimestamp,
			&lastFinishedAt,
			&payloadJSON,
		); err != nil {
			return nil, err
		}

		var payload any
		if len(payloadJSON) > 0 {
			if err := json.Unmarshal(payloadJSON, &payload); err != nil {
				return nil, fmt.Errorf("unmarshal payload: %w", err)
			}
		}

		var interval *time.Duration
		if intervalSeconds.Valid {
			dur := time.Duration(intervalSeconds.Int64) * time.Second
			interval = &dur
		}

		var once *int64
		if onceTimestamp.Valid {
			once = &onceTimestamp.Int64
		}

		jobs = append(jobs, &entity.Job{
			ID:             id,
			Kind:           entity.JobKind(kind),
			Status:         entity.JobStatus(status),
			Interval:       interval,
			Once:           once,
			LastFinishedAt: lastFinishedAt,
			Payload:        payload,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return jobs, nil
}

func (r *JobsRepository) Upsert(ctx context.Context, jobs []*entity.Job) error {
	if len(jobs) == 0 {
		return nil
	}

	for _, job := range jobs {
		payloadJSON, err := json.Marshal(job.Payload)
		if err != nil {
			return err
		}

		var intervalSeconds *int64
		if job.Interval != nil {
			seconds := int64(job.Interval.Seconds())
			intervalSeconds = &seconds
		}

		_, err = r.pool.Exec(ctx, queryUpsert,
			job.ID,
			int(job.Kind),
			string(job.Status),
			intervalSeconds,
			job.Once,
			job.LastFinishedAt,
			payloadJSON,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *JobsRepository) Delete(ctx context.Context, jobID string) error {
	result, err := r.pool.Exec(ctx, queryDelete, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	return nil
}
