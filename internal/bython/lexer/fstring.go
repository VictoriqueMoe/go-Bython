package lexer

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"go-Bython/internal/bython/diagnostic"
	"go-Bython/internal/bython/token"
)

func (l *lexer) scanFStringBody(tok token.Token, q byte, triple, raw bool) *diagnostic.SyntaxError {
	text := l.text

	for l.pos < len(text) {
		c := text[l.pos]
		switch c {
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
		case '\n':
			if !triple {
				return l.unterminated(tok, triple)
			}
			l.pos++
			l.advanceLine(l.pos)
		case '{':
			if l.pos+1 < len(text) && text[l.pos+1] == '{' {
				l.pos += 2
				continue
			}

			open := l.token(token.KindLBrace, l.pos, l.pos+1)
			l.pos++
			if err := l.scanReplacementField(open, l.fdepth+1, q, triple); err != nil {
				return err
			}
		case '}':
			if l.pos+1 < len(text) && text[l.pos+1] == '}' {
				l.pos += 2
				continue
			}
			if l.pos+1 >= len(text) {
				l.starved = true
			}
			return diagnostic.At(l.src, diagnostic.ErrFStringSingleBrace, l.token(token.KindRBrace, l.pos, l.pos+1), msgFStringSingleBrace)
		case '\\':
			l.skipFStringEscape(raw)
		default:
			l.pos++
		}
	}

	l.starved = true

	return l.unterminated(tok, triple)
}

func (l *lexer) skipFStringEscape(raw bool) {
	text := l.text
	next := l.pos + 1

	if next >= len(text) {
		l.pos = next
		return
	}

	if text[next] == '{' || text[next] == '}' {
		l.pos = next
		return
	}

	if !raw && text[next] == 'N' && next+1 < len(text) && text[next+1] == '{' {
		end := strings.IndexAny(text[next+2:], "}\n")
		if end >= 0 && text[next+2+end] == '}' {
			l.pos = next + 2 + end + 1
			return
		}
		if end < 0 {
			l.starved = true
		}
	}

	l.skipEscapedByte()
}

func (l *lexer) scanReplacementField(open token.Token, depth int, q byte, triple bool) *diagnostic.SyntaxError {
	if depth > maxFStringDepth {
		return diagnostic.At(l.src, diagnostic.ErrFStringTooDeep, open, msgFStringTooDeep)
	}

	saved := l.fdepth
	l.fdepth = depth
	err := l.scanFieldContents(open, depth, q, triple)
	l.fdepth = saved

	return err
}

func (l *lexer) scanFieldContents(open token.Token, depth int, q byte, triple bool) *diagnostic.SyntaxError {
	text := l.text

	var stackBuf [8]token.Token
	stack := stackBuf[:0]

	for {
		l.skipSpace()
		if l.pos >= len(text) {
			l.starved = true
			return diagnostic.At(l.src, diagnostic.ErrFStringExpectingBrace, open, msgFStringExpectingBrace)
		}

		if len(stack) == 0 {
			switch text[l.pos] {
			case '}':
				l.pos++
				return nil
			case ':':
				l.pos++
				return l.scanFormatSpec(open, depth, q, triple)
			case '!':
				if l.pos+1 >= len(text) || text[l.pos+1] != '=' {
					return l.scanConversion(open, depth, q, triple)
				}
			}
		}

		var tok token.Token
		if err := l.scanToken(&tok); err != nil {
			if err.Code == diagnostic.ErrUnterminatedString || err.Code == diagnostic.ErrUnterminatedTriple {
				return diagnostic.At(l.src, diagnostic.ErrFStringExpectingBrace, open, msgFStringExpectingBrace)
			}
			return err
		}

		switch tok.Kind {
		case token.KindEOF:
			return diagnostic.At(l.src, diagnostic.ErrFStringExpectingBrace, open, msgFStringExpectingBrace)
		case token.KindLParen, token.KindLBrack, token.KindLBrace:
			stack = append(stack, tok)
		case token.KindRParen, token.KindRBrack, token.KindRBrace:
			closer := text[tok.Start]
			if len(stack) == 0 {
				return diagnostic.At(l.src, diagnostic.ErrUnmatchedClose, tok, fmt.Sprintf(diagnostic.MsgUnmatchedClose, closer))
			}

			opener := stack[len(stack)-1]
			if token.CloserFor(opener.Kind) != tok.Kind {
				return diagnostic.At(l.src, diagnostic.ErrFStringMismatch, tok, fmt.Sprintf(msgFStringMismatch, closer, text[opener.Start]))
			}
			stack = stack[:len(stack)-1]
		}
	}
}

func (l *lexer) scanConversion(open token.Token, depth int, q byte, triple bool) *diagnostic.SyntaxError {
	text := l.text
	conv := l.pos + 1

	if conv >= len(text) {
		l.starved = true
		return diagnostic.At(l.src, diagnostic.ErrFStringExpectingBrace, open, msgFStringExpectingBrace)
	}

	if c := text[conv]; c != 's' && c != 'r' && c != 'a' {
		r, _ := utf8.DecodeRuneInString(text[conv:])
		return diagnostic.At(l.src, diagnostic.ErrFStringConversion, l.token(token.KindName, conv, conv), fmt.Sprintf(msgFStringConversion, r))
	}

	l.pos = conv + 1
	if l.pos < len(text) && text[l.pos] == '}' {
		l.pos++
		return nil
	}

	if l.pos < len(text) && text[l.pos] == ':' {
		l.pos++
		return l.scanFormatSpec(open, depth, q, triple)
	}

	if l.pos >= len(text) {
		l.starved = true
	}

	return diagnostic.At(l.src, diagnostic.ErrFStringExpectingBrace, open, msgFStringExpectingBrace)
}

func (l *lexer) scanFormatSpec(open token.Token, depth int, q byte, triple bool) *diagnostic.SyntaxError {
	text := l.text

	for l.pos < len(text) {
		c := text[l.pos]
		switch c {
		case '{':
			inner := l.token(token.KindLBrace, l.pos, l.pos+1)
			l.pos++
			if err := l.scanReplacementField(inner, depth+1, q, triple); err != nil {
				return err
			}
		case '}':
			l.pos++
			return nil
		case '\n':
			if !triple {
				return diagnostic.At(l.src, diagnostic.ErrFStringExpectingBrace, open, msgFStringExpectingBrace)
			}
			l.pos++
			l.advanceLine(l.pos)
		case q:
			if !triple || l.atTripleClose(q) {
				return diagnostic.At(l.src, diagnostic.ErrFStringExpectingBrace, open, msgFStringExpectingBrace)
			}
			l.pos++
		default:
			l.pos++
		}
	}

	l.starved = true

	return diagnostic.At(l.src, diagnostic.ErrFStringExpectingBrace, open, msgFStringExpectingBrace)
}
