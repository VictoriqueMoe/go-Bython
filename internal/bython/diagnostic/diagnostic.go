package diagnostic

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"go-Bython/internal/bython/token"
)

type (
	ErrorCode uint8

	SyntaxError struct {
		Code    ErrorCode
		Line    int
		Col     int
		RelLine int
		RelCol  int
		Msg     string
	}
)

const (
	ErrUnterminatedString ErrorCode = iota + 1
	ErrUnterminatedTriple
	ErrFStringSingleBrace
	ErrFStringExpectingBrace
	ErrFStringConversion
	ErrFStringTooDeep
	ErrFStringMismatch
	ErrInvalidCharacter
	ErrInvalidUTF8
	ErrBadContinuation
	ErrContinuationAtEOF
	ErrUnmatchedClose
	ErrMismatchedClose
	ErrUnclosedBracket
	ErrUnclosedBlock
	ErrBlockAfterSimple
	ErrBraceInBrackets
	ErrSemicolonInBrackets
	ErrJuxtaposed
	ErrUnexpected
	ErrExpectedBlock
	ErrExpectedExpr
	ErrExpectedName
	ErrBareKeyword
	ErrOrphanClause
	ErrClauseOrder
	ErrTryWithoutHandler
	ErrMixedExcept
	ErrAsync
	ErrDecoratorTarget
	ErrCaseOutsideMatch
	ErrNotCaseInMatch
	ErrMatchWithoutCase
	ErrTooDeep
)

const (
	MsgUnmatchedClose = "unmatched '%c'"
)

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Msg)
}

func At(src *token.Source, code ErrorCode, tok token.Token, msg string) *SyntaxError {
	return &SyntaxError{
		Code: code,
		Line: int(tok.Line),
		Col:  Column(src, tok),
		Msg:  msg,
	}
}

func After(src *token.Source, code ErrorCode, tok token.Token, msg string) *SyntaxError {
	body := src.Text[tok.Start:tok.End]

	last := strings.LastIndexByte(body, '\n')
	if last < 0 {
		tok.Start = tok.End
		return At(src, code, tok, msg)
	}

	return &SyntaxError{
		Code: code,
		Line: int(tok.Line) + strings.Count(body, "\n"),
		Col:  utf8.RuneCountInString(body[last+1:]) + 1,
		Msg:  msg,
	}
}

func Related(src *token.Source, code ErrorCode, tok, related token.Token, msg string) *SyntaxError {
	err := At(src, code, tok, msg)
	err.RelLine = int(related.Line)
	err.RelCol = Column(src, related)

	return err
}

func Column(src *token.Source, tok token.Token) int {
	prefix := src.Text[src.LineOf(tok).Start:tok.Start]

	return utf8.RuneCountInString(prefix) + 1
}
