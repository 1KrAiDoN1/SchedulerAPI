package dto

// JobCreate представляет структуру для создания задачи
type JobCreate struct {
	Once     string `json:"once,omitempty"`
	Interval string `json:"interval,omitempty"`
	Payload  any    `json:"payload,omitempty"`
}

type JobDTO struct {
	ID             string  `json:"id"`
	Kind           int     `json:"kind"` // JobKind as int
	Status         string  `json:"status"`
	Interval       *string `json:"interval,omitempty"` // duration as string
	Once           *int64  `json:"once,omitempty"`
	LastFinishedAt int64   `json:"lastFinishedAt"`
	Payload        any     `json:"payload"`
}
