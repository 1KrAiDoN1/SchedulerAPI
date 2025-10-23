package entity

// Execution представляет структуру выполнения задачи
type Execution struct {
	ID         string `json:"id,omitempty"`
	JobID      string `json:"jobId,omitempty"`
	WorkerID   string `json:"workerId,omitempty"`
	Status     string `json:"status,omitempty"`
	StartedAt  int64  `json:"startedAt,omitempty"`
	FinishedAt int64  `json:"finishedAt,omitempty"`
}
