package tokentest

import (
	"fmt"
	"strings"

	"go-Bython/internal/bython/token"
)

func RoundTrip(s *token.Source) error {
	var b strings.Builder
	textLen := len(s.Text)
	prev := 0

	for i, tok := range s.Tokens {
		if tok.Start < prev || tok.End < tok.Start || tok.End > textLen {
			return fmt.Errorf("token %d [%d,%d) overlaps or is out of order (prev end %d)", i, tok.Start, tok.End, prev)
		}

		if err := checkGap(s.Text, prev, tok.Start); err != nil {
			return err
		}

		b.WriteString(s.Text[prev:tok.Start])
		b.WriteString(s.Text[tok.Start:tok.End])
		prev = tok.End
	}

	if last := s.Tokens[len(s.Tokens)-1]; last.Kind != token.KindEOF || last.Start != textLen {
		return fmt.Errorf("last token is %v at %d, want EOF at %d", last.Kind, last.Start, len(s.Text))
	}

	if b.String() != s.Text {
		return fmt.Errorf("round trip mismatch: %q != %q", b.String(), s.Text)
	}

	return nil
}

func checkGap(text string, from, to int) error {
	for i := from; i < to; i++ {
		switch text[i] {
		case ' ', '\t', '\f':
		case '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				return fmt.Errorf("gap at %d holds a CRLF", i)
			}
		default:
			return fmt.Errorf("gap at %d holds %q", i, text[i])
		}
	}

	return nil
}
