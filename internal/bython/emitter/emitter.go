package emitter

import (
	"strings"

	"go-Bython/internal/bython/ast"
	"go-Bython/internal/bython/token"
)

type (
	emitter struct {
		text   string
		toks   []token.Token
		unit   int
		spaces string
		out    strings.Builder
		buf    []byte
	}
)

const (
	defaultIndentSize = 4
	lineBufferSize    = 256
)

var (
	spacePool = strings.Repeat(" ", 256)
)

func Emit(m *ast.Module, indentSize int) string {
	if indentSize < 1 {
		indentSize = defaultIndentSize
	}

	text := m.Source.Text
	e := emitter{
		text:   text,
		toks:   m.Source.Tokens,
		unit:   indentSize,
		spaces: spacePool,
		buf:    make([]byte, 0, lineBufferSize),
	}
	e.out.Grow(len(text) + len(text)/4)

	e.stmts(m.Body, 0)

	return e.out.String()
}

func (e *emitter) indent(level int) {
	width := level * e.unit
	if width > len(e.spaces) {
		e.spaces = strings.Repeat(" ", 2*width)
	}

	e.buf = append(e.buf, e.spaces[:width]...)
}

func (e *emitter) flush() {
	end := len(e.buf)
	for end > 0 && (e.buf[end-1] == ' ' || e.buf[end-1] == '\t') {
		end--
	}

	e.out.Write(e.buf[:end])
	e.out.WriteByte('\n')
	e.buf = e.buf[:0]
}

func (e *emitter) stmts(body []ast.Stmt, level int) {
	for _, s := range body {
		switch st := s.(type) {
		case *ast.BlankStmt:
			e.flush()
		case *ast.CommentStmt:
			e.indent(level)
			e.appendToken(e.toks[st.Tok])
			e.flush()
		case *ast.SimpleLine:
			e.simpleLine(st, level)
		case *ast.CompoundStmt:
			e.compound(st, level)
		}
	}
}

func (e *emitter) simpleLine(sl *ast.SimpleLine, level int) {
	e.indent(level)
	base := e.toks[sl.Stmts[0].Expr.Span.First].Indent

	for i, st := range sl.Stmts {
		if i > 0 {
			prev := sl.Stmts[i-1]
			separator := ast.Span{First: prev.Expr.Span.End, End: prev.FirstSemi + 1}
			e.span(separator, level, base, e.toks[prev.Expr.Span.End-1].End)
			e.appendGap(e.toks[prev.LastSemi].End, e.toks[st.Expr.Span.First].Start)
		}
		e.span(st.Expr.Span, level, base, -1)
	}

	e.trailing(sl.Trailing)
	e.flush()
}

func (e *emitter) compound(cs *ast.CompoundStmt, level int) {
	for _, d := range cs.Decorators {
		e.stmts(d.Leading, level)
		e.indent(level)
		e.span(ast.Span{First: d.At, End: d.Expr.Span.End}, level, e.toks[d.At].Indent, -1)
		e.trailing(d.Trailing)
		e.flush()
	}

	for _, c := range cs.Clauses {
		e.stmts(c.Leading, level)
		e.indent(level)
		e.span(c.Header.Span, level, e.toks[c.Header.Span.First].Indent, -1)
		e.buf = append(e.buf, ':')
		e.flush()

		e.stmts(c.Block.Body, level+1)

		if !hasCode(c.Block.Body) {
			e.indent(level + 1)
			e.buf = append(e.buf, "pass"...)
			e.flush()
		}
	}
}

func (e *emitter) trailing(idx int) {
	if idx < 0 {
		return
	}

	comment := e.toks[idx]
	gapStart := e.toks[idx-1].End
	if gapStart == comment.Start {
		e.buf = append(e.buf, ' ')
	} else {
		e.appendGap(gapStart, comment.Start)
	}

	e.appendToken(comment)
}

func (e *emitter) span(sp ast.Span, level, base, prevEnd int) {
	lineStart := false
	afterComment := false

	for _, tok := range e.toks[sp.First:sp.End] {
		switch tok.Kind {
		case token.KindNewline:
			e.flush()
			lineStart = true
			afterComment = false
			prevEnd = -1
			continue
		case token.KindLineJoin:
			if afterComment {
				e.flush()
			} else {
				if prevEnd >= 0 {
					e.appendGap(prevEnd, tok.Start)
				}
				e.buf = append(e.buf, '\\')
				e.flush()
			}

			lineStart = true
			afterComment = false
			prevEnd = -1
			continue
		}

		if afterComment {
			e.flush()
			lineStart = true
		}

		if lineStart {
			e.indent(level + rescale(tok.Indent-base, e.unit))
			lineStart = false
		} else if prevEnd >= 0 {
			e.appendGap(prevEnd, tok.Start)
		}

		e.appendToken(tok)
		prevEnd = tok.End
		afterComment = tok.Kind == token.KindComment
	}
}

func (e *emitter) appendGap(from, to int) {
	switch to - from {
	case 0:
		return
	case 1:
		if c := e.text[from]; c != '\r' {
			e.buf = append(e.buf, c)
			return
		}
	}

	gap := e.text[from:to]
	if strings.IndexByte(gap, '\r') < 0 {
		e.buf = append(e.buf, gap...)
		return
	}

	start := len(e.buf)
	for _, c := range []byte(gap) {
		if c != '\r' {
			e.buf = append(e.buf, c)
		}
	}

	if len(e.buf) == start {
		e.buf = append(e.buf, ' ')
	}
}

func (e *emitter) appendToken(tok token.Token) {
	text := e.text[tok.Start:tok.End]
	if tok.Flags&token.FlagMultiline == 0 || strings.IndexByte(text, '\r') < 0 {
		e.buf = append(e.buf, text...)
		return
	}

	for {
		before, after, found := strings.Cut(text, "\r\n")
		e.buf = append(e.buf, before...)
		if !found {
			return
		}
		e.buf = append(e.buf, '\n')
		text = after
	}
}

func rescale(rel, unit int) int {
	if rel < 0 {
		return 0
	}

	levels := rel / unit
	if rel > 0 && levels == 0 {
		levels = 1
	}

	return levels
}

func hasCode(body []ast.Stmt) bool {
	for _, s := range body {
		switch s.(type) {
		case *ast.SimpleLine, *ast.CompoundStmt:
			return true
		}
	}

	return false
}
