package lexer

import (
	"io"
	"math"
	"strings"

	"go-Bython/internal/bython/token"
)

type (
	lineLead uint8

	Stream struct {
		l              lexer
		pending        []token.Token
		depth          int
		lead           lineLead
		decorated      bool
		mayContinue    bool
		whole          bool
		compactPending bool
	}
)

const (
	leadNone lineLead = iota
	leadCode
	leadDecorator
)

func (st *Stream) Reset(s *token.Source, r io.Reader) {
	st.l.start(s, r, "")
	st.clear()
}

func (st *Stream) ResetString(s *token.Source, src string) error {
	text := strings.TrimPrefix(src, byteOrderMark)
	if len(text) > math.MaxInt32 {
		return ErrInputTooLarge
	}

	st.l.start(s, nil, text)
	st.clear()

	return nil
}

func (st *Stream) Release() {
	st.l.text = ""
	st.l.r = nil
	st.l.src = nil
	st.pending = st.pending[:0]

	if cap(st.l.block) > maxPooledBlock {
		st.l.block = nil
	}
}

func (st *Stream) clear() {
	st.pending = st.pending[:0]
	st.depth = 0
	st.lead = leadNone
	st.decorated = false
	st.mayContinue = false
	st.whole = false
	st.compactPending = false
}

func (st *Stream) Next() (bool, error) {
	if st.compactPending {
		st.compact()
	}

	l := &st.l
	s := l.src

	for {
		var tok token.Token
		if err := l.next(&tok); err != nil {
			return false, err
		}

		if tok.Kind == token.KindEOF {
			l.finish(tok)
			return true, nil
		}

		mayContinue := st.mayContinue
		st.track(tok)
		s.Tokens = append(s.Tokens, tok)

		if tok.Kind != token.KindNewline || st.whole || st.depth != 0 || st.decorated {
			continue
		}

		boundary, done, err := st.lookahead(tok, mayContinue)
		if err != nil {
			return false, err
		}

		if boundary || done {
			st.compactPending = boundary
			return done, nil
		}
	}
}

func (st *Stream) lookahead(newline token.Token, skipTrivia bool) (bool, bool, error) {
	l := &st.l
	s := l.src
	st.pending = st.pending[:0]

	for {
		var tok token.Token
		if err := l.next(&tok); err != nil {
			return false, false, err
		}

		if tok.Kind == token.KindEOF {
			s.Tokens = append(s.Tokens, st.pending...)
			st.pending = st.pending[:0]
			l.finish(tok)
			return false, true, nil
		}

		st.track(tok)
		st.pending = append(st.pending, tok)

		if skipTrivia {
			switch tok.Kind {
			case token.KindNewline, token.KindComment, token.KindLineJoin:
				continue
			}
		}

		if continuesStatement(tok) {
			s.Tokens = append(s.Tokens, st.pending...)
			st.pending = st.pending[:0]
			return false, false, nil
		}

		s.Tokens = append(s.Tokens, token.Token{
			Kind:  token.KindEOF,
			Start: newline.End,
			End:   newline.End,
			Line:  newline.Line + 1,
		})

		return true, false, nil
	}
}

func (st *Stream) track(tok token.Token) {
	switch tok.Kind {
	case token.KindNewline:
		if st.depth != 0 {
			return
		}

		switch st.lead {
		case leadDecorator:
			st.decorated = true
		case leadCode:
			st.decorated = false
		}
		st.lead = leadNone
		st.mayContinue = false

		return
	case token.KindLParen, token.KindLBrack, token.KindLBrace:
		st.depth++
	case token.KindRParen, token.KindRBrack, token.KindRBrace:
		st.depth--
	}

	st.l.lineHasToken = true

	if tok.Kind == token.KindRBrace || st.opensClause(tok) {
		st.mayContinue = true
	}

	if st.lead != leadNone || tok.Kind == token.KindComment || tok.Kind == token.KindLineJoin {
		return
	}

	st.lead = leadCode
	if tok.Kind == token.KindOperator && st.l.text[tok.Start:tok.End] == "@" {
		st.lead = leadDecorator
	}
}

func (st *Stream) compact() {
	l := &st.l
	s := l.src
	st.compactPending = false

	first := int32(l.line)
	if len(st.pending) > 0 {
		first = st.pending[0].Line
	}

	keep := first - s.FirstLine
	cut := s.Lines[keep].Start

	s.Tokens = append(s.Tokens[:0], st.pending...)
	st.pending = st.pending[:0]
	for i := range s.Tokens {
		s.Tokens[i].Start -= cut
		s.Tokens[i].End -= cut
	}

	n := copy(s.Lines, s.Lines[keep:])
	s.Lines = s.Lines[:n]
	for i := range s.Lines {
		s.Lines[i].Start -= cut
	}
	s.FirstLine = first

	shift := int(cut)
	l.text = l.text[shift:]
	s.Text = l.text
	l.pos -= shift
	l.lineStart -= shift
	l.lastNL -= shift
}

func (st *Stream) opensClause(tok token.Token) bool {
	switch tok.Kind {
	case token.KindKeyword:
		switch tok.Kw {
		case token.KwIf, token.KwElif, token.KwElse, token.KwWhile, token.KwFor, token.KwTry, token.KwExcept, token.KwFinally, token.KwWith, token.KwDef, token.KwClass, token.KwAsync:
			return true
		}
	case token.KindName:
		word := st.l.text[tok.Start:tok.End]
		return word == "match" || word == "case"
	}

	return false
}

func continuesStatement(tok token.Token) bool {
	switch tok.Kind {
	case token.KindLBrace:
		return true
	case token.KindKeyword:
		switch tok.Kw {
		case token.KwElif, token.KwElse, token.KwExcept, token.KwFinally:
			return true
		}
	}

	return false
}
