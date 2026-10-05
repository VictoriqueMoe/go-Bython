//go:build !windows

package processor

import "time"

type (
	stopwatch struct {
		start time.Time
	}
)

func startStopwatch() stopwatch {
	return stopwatch{start: time.Now()}
}

func (s stopwatch) elapsed() time.Duration {
	return time.Since(s.start)
}
