package bython

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"

	"go-Bython/internal/bython/ast"
	"go-Bython/internal/bython/emitter"
	"go-Bython/internal/bython/lexer"
	"go-Bython/internal/bython/parser"
	"go-Bython/internal/bython/token"
)

type (
	translator struct {
		stream  lexer.Stream
		source  token.Source
		module  ast.Module
		parser  parser.Parser
		emitter emitter.Emitter
		out     *bufio.Writer
	}
)

const (
	outputBufferSize = 64 << 10
)

var (
	translators = sync.Pool{New: func() any {
		return &translator{out: bufio.NewWriterSize(nil, outputBufferSize)}
	}}
)

func Translate(src string, indentSize int) (string, error) {
	t := translators.Get().(*translator)
	defer t.release()

	if err := t.stream.ResetString(&t.source, src); err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(len(src) + len(src)/4)
	t.out.Reset(&b)

	if err := t.run(indentSize); err != nil {
		return "", err
	}

	return b.String(), nil
}

func TranslateStream(r io.Reader, w io.Writer, indentSize int) error {
	t := translators.Get().(*translator)
	defer t.release()

	t.stream.Reset(&t.source, r)
	t.out.Reset(w)

	return t.run(indentSize)
}

func (t *translator) run(indentSize int) error {
	for {
		done, err := t.stream.Next()
		if err != nil {
			return err
		}

		if err := t.parser.Parse(&t.module, &t.source); err != nil {
			return err
		}

		if err := t.emitter.Emit(t.out, &t.module, indentSize); err != nil {
			return fmt.Errorf("error writing output: %w", err)
		}

		if done {
			break
		}
	}

	if err := t.out.Flush(); err != nil {
		return fmt.Errorf("error writing output: %w", err)
	}

	return nil
}

func (t *translator) release() {
	t.stream.Release()
	t.out.Reset(nil)
	t.source.Text = ""
	t.module = ast.Module{}
	translators.Put(t)
}
