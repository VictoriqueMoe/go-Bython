package processor

import (
	"io"
	"time"
)

type (
	Timings struct {
		Open      time.Duration
		Read      time.Duration
		Translate time.Duration
		Write     time.Duration
		Finalise  time.Duration
		Total     time.Duration
	}

	TimingStep struct {
		Name     string
		Duration time.Duration
	}

	FolderSummary struct {
		Files   int
		Timings Timings
		Wall    time.Duration
	}

	timedReader struct {
		r       io.Reader
		elapsed time.Duration
	}

	timedWriter struct {
		w       io.Writer
		elapsed time.Duration
	}
)

func (t Timings) Steps() []TimingStep {
	return []TimingStep{
		{Name: "open", Duration: t.Open},
		{Name: "read", Duration: t.Read},
		{Name: "translate", Duration: t.Translate},
		{Name: "write", Duration: t.Write},
		{Name: "finalise", Duration: t.Finalise},
		{Name: "total", Duration: t.Total},
	}
}

func (t *Timings) Add(other Timings) {
	t.Open += other.Open
	t.Read += other.Read
	t.Translate += other.Translate
	t.Write += other.Write
	t.Finalise += other.Finalise
	t.Total += other.Total
}

func (r *timedReader) Read(p []byte) (int, error) {
	watch := startStopwatch()
	n, err := r.r.Read(p)
	r.elapsed += watch.elapsed()

	return n, err
}

func (w *timedWriter) Write(p []byte) (int, error) {
	watch := startStopwatch()
	n, err := w.w.Write(p)
	w.elapsed += watch.elapsed()

	return n, err
}
