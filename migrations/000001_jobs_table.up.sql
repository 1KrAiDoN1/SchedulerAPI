CREATE TABLE jobs (
    id VARCHAR(255) PRIMARY KEY,
    kind INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    interval_seconds BIGINT,
    once_timestamp BIGINT,
    last_finished_at BIGINT NOT NULL DEFAULT 0,
    payload JSONB
);