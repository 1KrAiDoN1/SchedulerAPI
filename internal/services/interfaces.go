package services

type JobServiceInterface interface {
	// Define methods for job service here
}

// Проверка на наличие всех методов интерфейса у структуры
var _ JobServiceInterface = (*JobsService)(nil)
