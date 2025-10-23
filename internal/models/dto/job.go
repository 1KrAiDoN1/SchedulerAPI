package dto

// JobCreate представляет структуру для создания задачи
type JobCreate struct {
	Once     string      `json:"once,omitempty"`
	Interval string      `json:"interval,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
}

type JobDTO struct {
	Once           string      `json:"once,omitempty"`
	Interval       string      `json:"interval,omitempty"`
	Status         string      `json:"status"`
	CreatedAt      int64       `json:"createdAt"`
	LastFinishedAt int64       `json:"lastFinishedAt"`
	Payload        interface{} `json:"payload"`
}
