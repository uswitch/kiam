package server

import (
	"fmt"
)

var (
	// ErrPodNotFound returned when no pod found with a matching IP
	ErrPodNotFound = fmt.Errorf("no pod found")
)
