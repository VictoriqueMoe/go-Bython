package bython

import (
	"sync"

	"go-Bython/internal/bython/ast"
	"go-Bython/internal/bython/emitter"
	"go-Bython/internal/bython/lexer"
	"go-Bython/internal/bython/parser"
	"go-Bython/internal/bython/token"
)

type (
	translator struct {
		source token.Source
		module ast.Module
		parser parser.Parser
	}
)

var (
	translators = sync.Pool{New: func() any { return &translator{} }}
)

func Translate(src string, indentSize int) (string, error) {
	t := translators.Get().(*translator)
	defer func() {
		t.source.Text = ""
		t.module = ast.Module{}
		translators.Put(t)
	}()

	if err := lexer.LexInto(&t.source, src); err != nil {
		return "", err
	}

	if err := t.parser.Parse(&t.module, &t.source); err != nil {
		return "", err
	}

	return emitter.Emit(&t.module, indentSize), nil
}
