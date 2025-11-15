-- Таблица для хранения задач планировщика
-- kind: 1 = interval (периодическая), 2 = once (одноразовая)
-- status: queued, running, completed, failed
-- interval_seconds: интервал в секундах для периодических задач
-- once_timestamp: timestamp в миллисекундах для одноразовых задач
-- last_finished_at: timestamp последнего завершения в миллисекундах
CREATE TABLE jobs (
    id VARCHAR(255) PRIMARY KEY,
    kind INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    interval_seconds BIGINT,
    once_timestamp BIGINT,
    last_finished_at BIGINT NOT NULL DEFAULT 0,
    payload JSONB
);