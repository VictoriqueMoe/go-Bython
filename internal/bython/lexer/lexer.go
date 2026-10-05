package lexer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"go-Bython/internal/bython/diagnostic"
	"go-Bython/internal/bython/token"
)

type (
	lexer struct {
		src           *token.Source
		text          string
		pos           int
		line          int
		lineStart     int
		lineHasToken  bool
		fdepth        int
		starved       bool
		pendingIndent bool
		r             io.Reader
		block         []byte
		exhausted     bool
		bomChecked    bool
		lastNL        int
		total         int64
	}

	snapshot struct {
		pos           int
		line          int
		lineStart     int
		lines         int
		lineHasToken  bool
		pendingIndent bool
	}
)

const (
	byteOrderMark   = "\xef\xbb\xbf"
	maxFStringDepth = 150
	minBlock        = 64 << 10
	maxPooledBlock  = 1 << 20
	maxEmptyReads   = 100
)

const (
	msgUnterminatedString    = "unterminated string literal"
	msgUnterminatedTriple    = "unterminated triple-quoted string literal"
	msgFStringSingleBrace    = "f-string: single '}' is not allowed"
	msgFStringExpectingBrace = "f-string: expecting '}'"
	msgFStringConversion     = "f-string: invalid conversion character '%c': expected 's', 'r' or 'a'"
	msgFStringTooDeep        = "f-string: expressions nested too deeply"
	msgFStringMismatch       = "f-string: closing parenthesis '%c' does not match opening parenthesis '%c'"
	msgInvalidCharacter      = "invalid character '%c' (U+%04X)"
	msgInvalidUTF8           = "invalid UTF-8 byte 0x%02X"
	msgBadContinuation       = "unexpected character after line continuation character"
	msgContinuationAtEOF     = "unexpected end of file after line continuation character"
)

var (
	ErrInputTooLarge = fmt.Errorf("input exceeds %d bytes", math.MaxInt32)
)

func Lex(src string) (*token.Source, error) {
	s := &token.Source{}

	var st Stream
	if err := st.ResetString(s, src); err != nil {
		return nil, err
	}
	st.whole = true

	if _, err := st.Next(); err != nil {
		return nil, err
	}

	return s, nil
}

func (l *lexer) start(s *token.Source, r io.Reader, text string) {
	block := l.block
	if cap(block) > maxPooledBlock {
		block = nil
	}

	*l = lexer{
		src:        s,
		text:       text,
		line:       1,
		r:          r,
		block:      block,
		exhausted:  r == nil,
		bomChecked: r == nil,
		lastNL:     strings.LastIndexByte(text, '\n'),
	}

	s.Text = text
	s.Tokens = s.Tokens[:0]
	s.FirstLine = 1
	s.Lines = append(s.Lines[:0], token.LineInfo{Indent: l.measureIndent(0)})
}

func (l *lexer) next(tok *token.Token) error {
	if l.exhausted && !l.pendingIndent {
		if err := l.scanToken(tok); err != nil {
			return err
		}
		return nil
	}

	for {
		if err := l.ensureLine(); err != nil {
			return err
		}

		snap := l.save()
		l.starved = false
		err := l.scanToken(tok)

		if l.exhausted || !l.starved {
			if err != nil {
				return err
			}
			return nil
		}

		l.restore(snap)

		if err := l.refill(); err != nil {
			return err
		}
	}
}

func (l *lexer) save() snapshot {
	return snapshot{
		pos:           l.pos,
		line:          l.line,
		lineStart:     l.lineStart,
		lines:         len(l.src.Lines),
		lineHasToken:  l.lineHasToken,
		pendingIndent: l.pendingIndent,
	}
}

func (l *lexer) restore(snap snapshot) {
	l.pos = snap.pos
	l.line = snap.line
	l.lineStart = snap.lineStart
	l.src.Lines = l.src.Lines[:snap.lines]
	l.lineHasToken = snap.lineHasToken
	l.pendingIndent = snap.pendingIndent
	l.fdepth = 0
}

func (l *lexer) ensureLine() error {
	for l.pos > l.lastNL && !l.exhausted {
		if err := l.refill(); err != nil {
			return err
		}
	}

	if l.pendingIndent {
		l.src.Lines[len(l.src.Lines)-1].Indent = l.measureIndent(l.lineStart)
	}

	return nil
}

func (l *lexer) refill() error {
	want := max(minBlock, len(l.text))
	if cap(l.block) < want {
		l.block = make([]byte, want)
	}

	n, err := l.fill(l.block[:want])
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	l.total += int64(n)
	if l.total > math.MaxInt32 {
		return ErrInputTooLarge
	}

	data := l.block[:n]
	if !l.bomChecked && (len(data) >= len(byteOrderMark) || l.exhausted) {
		l.bomChecked = true
		if len(data) >= len(byteOrderMark) && string(data[:len(byteOrderMark)]) == byteOrderMark {
			data = data[len(byteOrderMark):]
		}
	}

	if len(data) == 0 {
		return nil
	}

	if idx := bytes.LastIndexByte(data, '\n'); idx >= 0 {
		l.lastNL = len(l.text) + idx
	}

	l.text += string(data)
	l.src.Text = l.text

	return nil
}

func (l *lexer) fill(buf []byte) (int, error) {
	n := 0
	empty := 0

	for n < len(buf) {
		m, err := l.r.Read(buf[n:])
		n += m

		if errors.Is(err, io.EOF) {
			l.exhausted = true
			return n, nil
		}

		if err != nil {
			return n, err
		}

		if m > 0 {
			empty = 0
			continue
		}

		empty++
		if empty >= maxEmptyReads {
			return n, io.ErrNoProgress
		}
	}

	return n, nil
}

func (l *lexer) finish(eof token.Token) {
	end := len(l.text)

	if l.lineHasToken {
		l.src.Tokens = append(l.src.Tokens, l.token(token.KindNewline, end, end))
	} else if end > l.lineStart {
		blank := l.token(token.KindNewline, end, end)
		blank.Flags = token.FlagBlank
		l.src.Tokens = append(l.src.Tokens, blank)
	}

	l.src.Tokens = append(l.src.Tokens, eof)
}

func (l *lexer) token(kind token.Kind, start, end int) token.Token {
	return token.Token{
		Kind:  kind,
		Start: int32(start),
		End:   int32(end),
		Line:  int32(l.line),
	}
}

func (l *lexer) measureIndent(at int) int32 {
	width := 0
	for at+width < len(l.text) {
		c := l.text[at+width]
		if c != ' ' && c != '\t' && c != '\f' {
			break
		}
		width++
	}

	l.pendingIndent = at+width == len(l.text) && !l.exhausted

	return int32(width)
}

func (l *lexer) advanceLine(at int) {
	l.line++
	l.lineStart = at
	l.src.Lines = append(l.src.Lines, token.LineInfo{Start: int32(at), Indent: l.measureIndent(at)})
}

func (l *lexer) skipSpace() {
	text := l.text
	for l.pos < len(text) {
		c := text[l.pos]
		if c == ' ' || c == '\t' || c == '\f' {
			l.pos++
			continue
		}

		if c == '\r' && !l.isCRLF(l.pos) {
			l.pos++
			continue
		}

		return
	}
}

func (l *lexer) isCRLF(at int) bool {
	return at+1 < len(l.text) && l.text[at] == '\r' && l.text[at+1] == '\n'
}

func (l *lexer) scanToken(tok *token.Token) *diagnostic.SyntaxError {
	l.skipSpace()

	text := l.text
	start := l.pos
	if start >= len(text) {
		l.starved = true
		*tok = l.token(token.KindEOF, start, start)
		return nil
	}

	c := text[start]
	switch {
	case c == '\n' || c == '\r':
		l.newline(tok, start)
		return nil
	case c == '#':
		l.comment(tok, start)
		return nil
	case c == '\\':
		return l.lineJoin(tok, start)
	case isNameStart(c):
		return l.scanName(tok, start)
	case isDigit(c) || (c == '.' && start+1 < len(text) && isDigit(text[start+1])):
		l.scanNumber(tok, start)
		return nil
	case c == '"' || c == '\'':
		return l.scanString(tok, start, start)
	}

	return l.scanOperator(tok, start)
}

func (l *lexer) newline(tok *token.Token, start int) {
	end := start + 1
	if l.text[start] == '\r' {
		end = start + 2
	}

	*tok = l.token(token.KindNewline, start, end)
	if !l.lineHasToken {
		tok.Flags = token.FlagBlank
	}

	l.pos = end
	l.advanceLine(end)
	l.lineHasToken = false
}

func (l *lexer) comment(tok *token.Token, start int) {
	text := l.text

	end := start
	for end < len(text) && text[end] != '\n' && text[end] != '\r' {
		end++
	}

	l.pos = end
	*tok = l.token(token.KindComment, start, end)
}

func (l *lexer) lineJoin(tok *token.Token, start int) *diagnostic.SyntaxError {
	text := l.text
	next := start + 1

	end := -1
	switch {
	case next < len(text) && text[next] == '\n':
		end = next + 1
	case l.isCRLF(next):
		end = next + 2
	case next >= len(text) || (text[next] == '\r' && next+1 >= len(text)):
		l.starved = true
		return diagnostic.At(l.src, diagnostic.ErrContinuationAtEOF, l.token(token.KindLineJoin, start, start), msgContinuationAtEOF)
	default:
		return diagnostic.At(l.src, diagnostic.ErrBadContinuation, l.token(token.KindLineJoin, start, start), msgBadContinuation)
	}

	*tok = l.token(token.KindLineJoin, start, end)
	l.pos = end
	l.lineHasToken = true
	l.advanceLine(end)

	return nil
}

func (l *lexer) scanName(tok *token.Token, start int) *diagnostic.SyntaxError {
	text := l.text
	pos := start

	for pos < len(text) {
		c := text[pos]
		if c < utf8.RuneSelf {
			if isNameStart(c) || (pos > start && isDigit(c)) {
				pos++
				continue
			}
			break
		}

		r, size := utf8.DecodeRuneInString(text[pos:])
		if r == utf8.RuneError && size == 1 {
			if !utf8.FullRuneInString(text[pos:]) {
				l.starved = true
			}
			if pos == start {
				return diagnostic.At(l.src, diagnostic.ErrInvalidUTF8, l.token(token.KindName, pos, pos), fmt.Sprintf(msgInvalidUTF8, c))
			}
			break
		}

		if unicode.IsLetter(r) || (pos > start && unicode.In(r, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc)) {
			pos += size
			continue
		}

		if pos == start {
			return diagnostic.At(l.src, diagnostic.ErrInvalidCharacter, l.token(token.KindName, pos, pos), fmt.Sprintf(msgInvalidCharacter, r, r))
		}
		break
	}

	if pos < len(text) && (text[pos] == '"' || text[pos] == '\'') && token.IsStringPrefix(text[start:pos]) {
		return l.scanString(tok, start, pos)
	}

	l.pos = pos

	*tok = l.token(token.KindName, start, pos)
	if kw := token.Lookup(text[start:pos]); kw != token.NoKeyword {
		tok.Kind = token.KindKeyword
		tok.Kw = kw
	}

	return nil
}

func (l *lexer) scanNumber(tok *token.Token, start int) {
	text := l.text
	pos := start

	if text[pos] == '0' && pos+1 < len(text) && strings.IndexByte("xXoObB", text[pos+1]) >= 0 {
		pos += 2
		for pos < len(text) && (isHexDigit(text[pos]) || text[pos] == '_') {
			pos++
		}
	} else {
		pos = l.skipDigits(pos)
		if pos < len(text) && text[pos] == '.' {
			pos = l.skipDigits(pos + 1)
		}

		if pos < len(text) && (text[pos] == 'e' || text[pos] == 'E') {
			exp := pos + 1
			if exp < len(text) && (text[exp] == '+' || text[exp] == '-') {
				exp++
			}
			if exp < len(text) && (isDigit(text[exp]) || text[exp] == '_') {
				pos = l.skipDigits(exp)
			}
		}

		if pos < len(text) && (text[pos] == 'j' || text[pos] == 'J') {
			pos++
		}
	}

	l.pos = pos
	*tok = l.token(token.KindNumber, start, pos)
}

func (l *lexer) skipDigits(pos int) int {
	for pos < len(l.text) && (isDigit(l.text[pos]) || l.text[pos] == '_') {
		pos++
	}

	return pos
}

func (l *lexer) scanOperator(tok *token.Token, start int) *diagnostic.SyntaxError {
	rest := l.text[start:]

	kind, size := token.LookupOperator(rest)
	if size == 0 {
		if len(rest) == 1 {
			l.starved = true
		}

		r, _ := utf8.DecodeRuneInString(rest)
		return diagnostic.At(l.src, diagnostic.ErrInvalidCharacter, l.token(token.KindOperator, start, start), fmt.Sprintf(msgInvalidCharacter, r, r))
	}

	l.pos = start + size
	*tok = l.token(kind, start, l.pos)

	return nil
}

func (l *lexer) scanString(tok *token.Token, start, quote int) *diagnostic.SyntaxError {
	text := l.text
	*tok = l.token(token.KindString, start, start)

	raw := false
	for _, p := range []byte(text[start:quote]) {
		switch p | 0x20 {
		case 'r':
			raw = true
		case 'f', 't':
			tok.Flags |= token.FlagFString
		}
	}

	q := text[quote]
	l.pos = quote + 1

	triple := false
	if l.pos+1 < len(text) && text[l.pos] == q && text[l.pos+1] == q {
		triple = true
		tok.Flags |= token.FlagTriple
		l.pos += 2
	}

	var err *diagnostic.SyntaxError
	if tok.Flags&token.FlagFString != 0 {
		err = l.scanFStringBody(*tok, q, triple, raw)
	} else {
		err = l.scanPlainBody(*tok, q, triple)
	}
	if err != nil {
		return err
	}

	tok.End = int32(l.pos)
	if strings.IndexByte(text[start:l.pos], '\n') >= 0 {
		tok.Flags |= token.FlagMultiline
	}

	return nil
}

func (l *lexer) scanPlainBody(tok token.Token, q byte, triple bool) *diagnostic.SyntaxError {
	text := l.text

	for l.pos < len(text) {
		c := text[l.pos]
		switch c {
		case '\\':
			l.skipEscapedByte()
		case '\n':
			if !triple {
				return l.unterminated(tok, triple)
			}
			l.pos++
			l.advanceLine(l.pos)
		case q:
			if !triple {
				l.pos++
				return nil
			}
			if l.atTripleClose(q) {
				l.pos += 3
				return nil
			}
			l.pos++
		default:
			l.pos++
		}
	}

	l.starved = true

	return l.unterminated(tok, triple)
}

func (l *lexer) skipEscapedByte() {
	text := l.text
	next := l.pos + 1

	switch {
	case next >= len(text):
		l.pos = next
	case text[next] == '\n':
		l.pos = next + 1
		l.advanceLine(l.pos)
	case l.isCRLF(next):
		l.pos = next + 2
		l.advanceLine(l.pos)
	default:
		l.pos = next + 1
	}
}

func (l *lexer) atTripleClose(q byte) bool {
	text := l.text

	return l.pos+2 < len(text) && text[l.pos] == q && text[l.pos+1] == q && text[l.pos+2] == q
}

func (l *lexer) unterminated(tok token.Token, triple bool) *diagnostic.SyntaxError {
	if triple {
		return diagnostic.At(l.src, diagnostic.ErrUnterminatedTriple, tok, msgUnterminatedTriple)
	}

	return diagnostic.At(l.src, diagnostic.ErrUnterminatedString, tok, msgUnterminatedString)
}

func isNameStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c >= utf8.RuneSelf
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
