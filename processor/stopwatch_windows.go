package processor

import (
	"syscall"
	"time"
	"unsafe"
)

type (
	stopwatch struct {
		start int64
	}
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	queryPerformanceCounter   = kernel32.NewProc("QueryPerformanceCounter")
	queryPerformanceFrequency = kernel32.NewProc("QueryPerformanceFrequency")

	performanceFrequency = func() int64 {
		var frequency int64
		queryPerformanceFrequency.Call(uintptr(unsafe.Pointer(&frequency)))

		return frequency
	}()
)

func performanceCounter() int64 {
	var ticks int64
	queryPerformanceCounter.Call(uintptr(unsafe.Pointer(&ticks)))

	return ticks
}

func startStopwatch() stopwatch {
	return stopwatch{start: performanceCounter()}
}

func (s stopwatch) elapsed() time.Duration {
	ticks := performanceCounter() - s.start

	whole := ticks / performanceFrequency * int64(time.Second)
	fraction := ticks % performanceFrequency * int64(time.Second) / performanceFrequency

	return time.Duration(whole + fraction)
}
