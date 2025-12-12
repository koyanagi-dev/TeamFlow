package task

import "errors"

// Sentinel errors used by task usecases.
var (
	ErrInvalidInput = errors.New("invalid input")
	ErrTaskNotFound = errors.New("task not found")
)
