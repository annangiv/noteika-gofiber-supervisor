package utils

import (
	"runtime"
	"time"
)

type HealthStatus struct {
	Status     string  `json:"status"`
	Uptime     string  `json:"uptime"`
	Goroutines int     `json:"goroutines"`
	MemoryMB   float64 `json:"memory_mb"`
}

var startTime = time.Now()

func GetHealthStatus() HealthStatus {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return HealthStatus{
		Status:     "healthy",
		Uptime:     time.Since(startTime).String(),
		Goroutines: runtime.NumGoroutine(),
		MemoryMB:   float64(m.Alloc) / 1024 / 1024,
	}
}
