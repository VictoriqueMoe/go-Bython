package processor

import (
	"errors"
	"fmt"
	"io"
	"os"

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
	outputFileMode    = 0o666
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
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	output, err := bython.Translate(string(data), p.indentSize)
	if err != nil {
		return err
	}

	_, err = io.WriteString(writer, output)

	return err
}

func (p *PythonPreprocessor) ProcessFile(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}

	output, err := bython.Translate(string(data), p.indentSize)
	if err != nil {
		if _, ok := errors.AsType[*diagnostic.SyntaxError](err); ok {
			return fmt.Errorf("%s:%w", inputPath, err)
		}
		return err
	}

	if err := os.WriteFile(outputPath, []byte(output), outputFileMode); err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}

	return nil
}

func (p *PythonPreprocessor) IndentSize() int {
	return p.indentSize
}
