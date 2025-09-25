package syntax

import "go-Bython/syntax/engines"

// Engine defines the interface for syntax validation engines
type Engine interface {
	CheckSyntax(line string, lineNumber int) error

	Reset()
}

// Factory creates and manages syntax validation engines
type Factory struct {
	engines []Engine
}

// NewFactory creates a new factory with default engines
func NewFactory() *Factory {
	return &Factory{
		engines: []Engine{
			engines.NewMixedStyleEngine(),
		},
	}
}

// CheckAllEngines runs all syntax engines on a line and returns the first error found
func (f *Factory) CheckAllEngines(line string, lineNumber int) error {
	for _, engine := range f.engines {
		if err := engine.CheckSyntax(line, lineNumber); err != nil {
			return err
		}
	}
	return nil
}

// ResetAllEngines clears state in all engines (called before processing a new file)
func (f *Factory) ResetAllEngines() {
	for _, engine := range f.engines {
		engine.Reset()
	}
}
