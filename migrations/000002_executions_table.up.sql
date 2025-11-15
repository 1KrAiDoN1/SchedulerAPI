-- Таблица для хранения истории выполнения задач
-- Хранит информацию о каждом запуске задачи воркером
-- status: queued, running, completed, failed, success
-- started_at, finished_at: timestamps в миллисекундах
CREATE TABLE executions (
    id VARCHAR(255) PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    worker_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at BIGINT NOT NULL,
    finished_at BIGINT,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

-- Индексы для быстрого поиска по job_id и worker_id
CREATE INDEX idx_executions_job_id ON executions(job_id);
CREATE INDEX idx_executions_worker_id ON executions(worker_id);

