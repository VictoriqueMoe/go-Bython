package processor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go-Bython/internal/bython"
	"go-Bython/internal/bython/diagnostic"
)

type (
	PythonPreprocessor struct {
		indentSize int
	}
)

const (
	defaultIndentSize = 4
	outputFileMode    = 0o644
	tempFilePattern   = ".*.tmp"
)

func NewPythonPreprocessor(indentSize int) Processor {
	if indentSize < 1 {
		indentSize = defaultIndentSize
	}

	return &PythonPreprocessor{indentSize: indentSize}
}

func (p *PythonPreprocessor) ProcessString(input string) (string, error) {
	output, err := bython.Translate(input, p.indentSize)
	if err != nil {
		return "", err
	}

	return output, nil
}

func (p *PythonPreprocessor) ProcessReader(reader io.Reader, writer io.Writer) error {
	return bython.TranslateStream(reader, writer, p.indentSize)
}

func (p *PythonPreprocessor) ProcessFile(inputPath, outputPath string) (Timings, error) {
	var timings Timings
	total := startStopwatch()

	input, err := os.Open(inputPath)
	if err != nil {
		return timings, fmt.Errorf("error opening input file: %w", err)
	}

	output, err := os.CreateTemp(filepath.Dir(outputPath), filepath.Base(outputPath)+tempFilePattern)
	if err != nil {
		err = fmt.Errorf("error creating output file: %w", err)
		if closeErr := input.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("error closing input file: %w", closeErr))
		}
		return timings, err
	}
	timings.Open = total.elapsed()

	if err := p.writeOutput(input, output, inputPath, outputPath, &timings); err != nil {
		if removeErr := os.Remove(output.Name()); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return timings, errors.Join(err, fmt.Errorf("error removing temporary output file: %w", removeErr))
		}
		return timings, err
	}

	timings.Total = total.elapsed()

	return timings, nil
}

func (p *PythonPreprocessor) writeOutput(input, output *os.File, inputPath, outputPath string, timings *Timings) error {
	reader := timedReader{r: input}
	writer := timedWriter{w: output}

	translate := startStopwatch()
	err := bython.TranslateStream(&reader, &writer, p.indentSize)
	timings.Read = reader.elapsed
	timings.Write = writer.elapsed
	timings.Translate = translate.elapsed() - reader.elapsed - writer.elapsed

	finalise := startStopwatch()
	inputErr := input.Close()
	closeErr := output.Close()

	if err != nil {
		if _, ok := errors.AsType[*diagnostic.SyntaxError](err); ok {
			err = fmt.Errorf("%s:%w", inputPath, err)
		}

		if inputErr != nil {
			err = errors.Join(err, fmt.Errorf("error closing input file: %w", inputErr))
		}

		if closeErr != nil {
			err = errors.Join(err, fmt.Errorf("error closing output file: %w", closeErr))
		}

		return err
	}

	if inputErr != nil {
		return fmt.Errorf("error closing input file: %w", inputErr)
	}

	if closeErr != nil {
		return fmt.Errorf("error creating output file: %w", closeErr)
	}

	if err := os.Chmod(output.Name(), outputFileMode); err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}

	if err := os.Rename(output.Name(), outputPath); err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	timings.Finalise = finalise.elapsed()

	return nil
}

func (p *PythonPreprocessor) IndentSize() int {
	return p.indentSize
}
