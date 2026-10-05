package lexer

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-Bython/internal/bython/diagnostic"
	"go-Bython/internal/bython/token"
	"go-Bython/internal/bython/tokentest"
)

type (
	tokSpec struct {
		kind token.Kind
		text string
	}
)

var (
	lexCases = []struct {
		name string
		src  string
		want []tokSpec
	}{
		{"assignment", "x = 1", []tokSpec{{token.KindName, "x"}, {token.KindOperator, "="}, {token.KindNumber, "1"}, {token.KindNewline, ""}}},
		{"brackets_and_punctuation", "f(a)[b]{c},:;", []tokSpec{
			{token.KindName, "f"}, {token.KindLParen, "("}, {token.KindName, "a"}, {token.KindRParen, ")"},
			{token.KindLBrack, "["}, {token.KindName, "b"}, {token.KindRBrack, "]"},
			{token.KindLBrace, "{"}, {token.KindName, "c"}, {token.KindRBrace, "}"},
			{token.KindComma, ","}, {token.KindColon, ":"}, {token.KindSemicolon, ";"}, {token.KindNewline, ""},
		}},
		{"ellipsis_and_dot", "a.b ...", []tokSpec{{token.KindName, "a"}, {token.KindOperator, "."}, {token.KindName, "b"}, {token.KindEllipsis, "..."}, {token.KindNewline, ""}}},
		{"walrus_and_colon", "(n := 1): x", []tokSpec{
			{token.KindLParen, "("}, {token.KindName, "n"}, {token.KindOperator, ":="}, {token.KindNumber, "1"}, {token.KindRParen, ")"},
			{token.KindColon, ":"}, {token.KindName, "x"}, {token.KindNewline, ""},
		}},
		{"comment", "x # c {", []tokSpec{{token.KindName, "x"}, {token.KindComment, "# c {"}, {token.KindNewline, ""}}},
		{"comment_stops_at_lone_carriage_return", "x # a\rb\r\ny", []tokSpec{{token.KindName, "x"}, {token.KindComment, "# a"}, {token.KindName, "b"}, {token.KindNewline, "\r\n"}, {token.KindName, "y"}, {token.KindNewline, ""}}},
		{"comment_drops_final_carriage_return", "x # a\r", []tokSpec{{token.KindName, "x"}, {token.KindComment, "# a"}, {token.KindNewline, ""}}},
		{"blank_lines", "\n\nx\n", []tokSpec{{token.KindNewline, "\n"}, {token.KindNewline, "\n"}, {token.KindName, "x"}, {token.KindNewline, "\n"}}},
		{"crlf", "x\r\ny", []tokSpec{{token.KindName, "x"}, {token.KindNewline, "\r\n"}, {token.KindName, "y"}, {token.KindNewline, ""}}},
		{"lone_carriage_return_is_space", "x\ry", []tokSpec{{token.KindName, "x"}, {token.KindName, "y"}, {token.KindNewline, ""}}},
		{"line_join", "a \\\n b", []tokSpec{{token.KindName, "a"}, {token.KindLineJoin, "\\\n"}, {token.KindName, "b"}, {token.KindNewline, ""}}},
		{"line_join_crlf", "a \\\r\n b", []tokSpec{{token.KindName, "a"}, {token.KindLineJoin, "\\\r\n"}, {token.KindName, "b"}, {token.KindNewline, ""}}},
		{"soft_keywords_are_names", "match case type _", []tokSpec{{token.KindName, "match"}, {token.KindName, "case"}, {token.KindName, "type"}, {token.KindName, "_"}, {token.KindNewline, ""}}},
		{"number_then_keyword", "1if", []tokSpec{{token.KindNumber, "1"}, {token.KindKeyword, "if"}, {token.KindNewline, ""}}},
		{"number_then_else", "1else", []tokSpec{{token.KindNumber, "1"}, {token.KindKeyword, "else"}, {token.KindNewline, ""}}},
		{"prefix_like_name", "rb + fr", []tokSpec{{token.KindName, "rb"}, {token.KindOperator, "+"}, {token.KindName, "fr"}, {token.KindNewline, ""}}},
		{"unicode_name", "größe = ü", []tokSpec{{token.KindName, "größe"}, {token.KindOperator, "="}, {token.KindName, "ü"}, {token.KindNewline, ""}}},
		{"bom_is_skipped", "\xef\xbb\xbfx", []tokSpec{{token.KindName, "x"}, {token.KindNewline, ""}}},
		{"whitespace_only_final_line", "x\n  ", []tokSpec{{token.KindName, "x"}, {token.KindNewline, "\n"}, {token.KindNewline, ""}}},
		{"triple_string_with_newline", "s = '''a\n}'''", []tokSpec{{token.KindName, "s"}, {token.KindOperator, "="}, {token.KindString, "'''a\n}'''"}, {token.KindNewline, ""}}},
		{"string_with_escaped_quote", `"a\"b" 'c'`, []tokSpec{{token.KindString, `"a\"b"`}, {token.KindString, "'c'"}, {token.KindNewline, ""}}},
		{"raw_string_escaped_quote", `r"\"" x`, []tokSpec{{token.KindString, `r"\""`}, {token.KindName, "x"}, {token.KindNewline, ""}}},
		{"empty_strings", `'' ""`, []tokSpec{{token.KindString, "''"}, {token.KindString, `""`}, {token.KindNewline, ""}}},
		{"empty_triple_string", `""""""`, []tokSpec{{token.KindString, `""""""`}, {token.KindNewline, ""}}},
	}

	operators = []string{
		"**=", "//=", ">>=", "<<=",
		"**", "//", "<<", ">>", "<=", ">=", "==", "!=", "->", ":=",
		"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "@=",
		"+", "-", "*", "/", "%", "@", "&", "|", "^", "~", "<", ">", ".", "=",
	}

	numberForms = []string{
		"0", "42", "1_000", "0x1F", "0XfF", "0o17", "0O7", "0b101", "0B1_0",
		"3.14", ".5", "5.", "1e10", "1E-5", "1e+3", "2j", "1.5e+3J", "1_0.0_1e1_0",
	}

	prefixes = []string{"r", "u", "b", "br", "rb", "f", "fr", "rf", "t", "tr", "rt"}

	fstringCases = []struct {
		name string
		src  string
	}{
		{"plain_field", `f"{x}"`},
		{"escaped_braces", `f"{{x}}"`},
		{"named_unicode_escape", `f"\N{BULLET} {x}"`},
		{"escaped_backslash_then_field", `f"\\{x}"`},
		{"backslash_brace_still_field", `f"\{x}"`},
		{"raw_backslash_brace", `rf"\{x}"`},
		{"nested_field_in_spec", `f"{x:{w}.{p}}"`},
		{"conversion", `f"{x!r}"`},
		{"conversion_with_spec", `f"{x!s:>10}"`},
		{"not_equal_is_not_conversion", `f"{a!=b}"`},
		{"walrus_is_spec", `f"{x:=5}"`},
		{"self_documenting", `f"{x = }"`},
		{"nested_same_quotes", `f"{"a"}"`},
		{"nested_fstring", `f"{f'{x}'}"`},
		{"dict_in_field", `f"{ {'a': 1}['a'] }"`},
		{"brace_in_nested_string", `f"{'}'}"`},
		{"slice_in_field", `f"{a[1:2]}"`},
		{"lambda_in_parens", `f"{(lambda: 1)()}"`},
		{"comment_and_newline_in_field", "f'''{x # c\n}'''"},
		{"newline_in_field", "f\"{x\n+ y}\""},
		{"triple_with_newlines", "f'''a\n{x}\nb'''"},
		{"tstring", `t"{x}"`},
		{"upper_prefix", `F"{x}"`},
	}

	fstringErrorCases = []struct {
		name string
		src  string
		code diagnostic.ErrorCode
	}{
		{"single_close", `f"}"`, diagnostic.ErrFStringSingleBrace},
		{"unclosed_field", `f"{a"`, diagnostic.ErrFStringExpectingBrace},
		{"bad_conversion", `f"{x!z}"`, diagnostic.ErrFStringConversion},
		{"mismatched_paren", `f"{(x]}"`, diagnostic.ErrFStringMismatch},
		{"spec_hits_quote", `f"{x:abc"`, diagnostic.ErrFStringExpectingBrace},
		{"unterminated", `f"abc`, diagnostic.ErrUnterminatedString},
	}

	lexErrorCases = []struct {
		name string
		src  string
		code diagnostic.ErrorCode
	}{
		{"unterminated_string", `"abc`, diagnostic.ErrUnterminatedString},
		{"newline_in_string", "'a\nb'", diagnostic.ErrUnterminatedString},
		{"unterminated_triple", `"""abc`, diagnostic.ErrUnterminatedTriple},
		{"dollar", "x = $", diagnostic.ErrInvalidCharacter},
		{"question_mark", "x?", diagnostic.ErrInvalidCharacter},
		{"backtick", "`x`", diagnostic.ErrInvalidCharacter},
		{"lone_bang", "!x", diagnostic.ErrInvalidCharacter},
		{"non_letter_rune", "x = €", diagnostic.ErrInvalidCharacter},
		{"invalid_utf8", "x = \xff", diagnostic.ErrInvalidUTF8},
		{"invalid_utf8_after_name", "O\xa2Y", diagnostic.ErrInvalidUTF8},
		{"bad_continuation", "a \\ b", diagnostic.ErrBadContinuation},
		{"continuation_at_eof", "a \\", diagnostic.ErrContinuationAtEOF},
		{"continuation_cr_at_eof", "a \\\r", diagnostic.ErrContinuationAtEOF},
	}
)

func TestLexTokens(t *testing.T) {
	for _, c := range lexCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			src := c.src

			//when
			s, err := Lex(src)

			//then
			require.NoError(t, err)
			assert.Equal(t, c.want, specs(s))
			require.NoError(t, tokentest.RoundTrip(s))
		})
	}
}

func TestLexOperators(t *testing.T) {
	for _, op := range operators {
		t.Run(op, func(t *testing.T) {
			//given
			src := "a " + op + " b"

			//when
			s, err := Lex(src)

			//then
			require.NoError(t, err)
			assert.Equal(t, []tokSpec{{token.KindName, "a"}, {token.KindOperator, op}, {token.KindName, "b"}, {token.KindNewline, ""}}, specs(s))
		})
	}
}

func TestLexKeywords(t *testing.T) {
	for kw := token.KwFalse; kw <= token.KwYield; kw++ {
		word := kw.String()
		t.Run(word, func(t *testing.T) {
			//given
			src := word

			//when
			s, err := Lex(src)

			//then
			require.NoError(t, err)
			assert.Equal(t, token.KindKeyword, s.Tokens[0].Kind)
			assert.Equal(t, kw, s.Tokens[0].Kw)
			assert.Equal(t, kw, token.Lookup(word))
		})
	}
}

func TestLexNumbers(t *testing.T) {
	for _, num := range numberForms {
		t.Run(num, func(t *testing.T) {
			//given
			src := num

			//when
			s, err := Lex(src)

			//then
			require.NoError(t, err)
			assert.Equal(t, []tokSpec{{token.KindNumber, num}, {token.KindNewline, ""}}, specs(s))
		})
	}
}

func TestLexStringPrefixes(t *testing.T) {
	for _, prefix := range prefixes {
		for _, variant := range casePermutations(prefix) {
			for _, quote := range []string{`"`, `'`, `"""`, `'''`} {
				src := variant + quote + "x" + quote
				t.Run(src, func(t *testing.T) {
					//given
					fstring := strings.ContainsAny(variant, "fFtT")

					//when
					s, err := Lex(src)

					//then
					require.NoError(t, err)
					assert.Equal(t, []tokSpec{{token.KindString, src}, {token.KindNewline, ""}}, specs(s))
					assert.Equal(t, fstring, s.Tokens[0].Flags&token.FlagFString != 0)
					assert.Equal(t, len(quote) == 3, s.Tokens[0].Flags&token.FlagTriple != 0)
				})
			}
		}
	}
}

func TestLexBlankFlags(t *testing.T) {
	//given
	src := "x\n\n# c\n  \ny"

	//when
	s, err := Lex(src)

	//then
	require.NoError(t, err)
	var blanks []bool
	for _, tok := range s.Tokens {
		if tok.Kind == token.KindNewline {
			blanks = append(blanks, tok.Flags&token.FlagBlank != 0)
		}
	}
	assert.Equal(t, []bool{false, true, false, true, false}, blanks)
}

func TestLexIndentAndLines(t *testing.T) {
	//given
	src := "a\n\t  b\n  '''x\ny''' c"

	//when
	s, err := Lex(src)

	//then
	require.NoError(t, err)
	b := s.Tokens[2]
	assert.Equal(t, 2, b.Line)
	assert.Equal(t, 3, b.Indent)
	str := s.Tokens[4]
	assert.Equal(t, token.KindString, str.Kind)
	assert.Equal(t, 3, str.Line)
	assert.NotZero(t, str.Flags&token.FlagMultiline)
	c := s.Tokens[5]
	assert.Equal(t, 4, c.Line)
}

func TestLexFStrings(t *testing.T) {
	for _, c := range fstringCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			src := c.src

			//when
			s, err := Lex(src)

			//then
			require.NoError(t, err)
			assert.Equal(t, []tokSpec{{token.KindString, src}, {token.KindNewline, ""}}, specs(s))
			assert.NotZero(t, s.Tokens[0].Flags&token.FlagFString)
		})
	}
}

func TestLexFStringErrors(t *testing.T) {
	for _, c := range fstringErrorCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			src := c.src

			//when
			_, err := Lex(src)

			//then
			assert.Equal(t, c.code, errorCode(t, err))
		})
	}
}

func TestLexFStringDepth(t *testing.T) {
	for _, depth := range []int{150, 151} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			//given
			src := "x"
			for range depth {
				src = `f"{` + src + `}"`
			}

			//when
			_, err := Lex(src)

			//then
			if depth > maxFStringDepth {
				assert.Equal(t, diagnostic.ErrFStringTooDeep, errorCode(t, err))
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestLexErrors(t *testing.T) {
	for _, c := range lexErrorCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			src := c.src

			//when
			_, err := Lex(src)

			//then
			assert.Equal(t, c.code, errorCode(t, err))
		})
	}
}

func TestLexRoundTrip(t *testing.T) {
	inputs := []string{
		"if x {\r\n\tpass;\r\n}\r\n",
		"s = f'''{x!r:>{w}}\n'''  # c\n\n  \x0c\n",
		"a = 1 \r b \\\n  + 2",
		"\xef\xbb\xbfdef f() { return {1: 2} }",
		"x = 1 #c\ry = 2\r\n#d\r",
	}

	for i, input := range inputs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			//given
			src := input

			//when
			s, err := Lex(src)

			//then
			require.NoError(t, err)
			assert.NoError(t, tokentest.RoundTrip(s))
		})
	}
}

func specs(s *token.Source) []tokSpec {
	var out []tokSpec
	for _, tok := range s.Tokens {
		if tok.Kind == token.KindEOF {
			continue
		}
		out = append(out, tokSpec{tok.Kind, s.Text[tok.Start:tok.End]})
	}

	return out
}

func errorCode(t *testing.T, err error) diagnostic.ErrorCode {
	t.Helper()

	se, ok := errors.AsType[*diagnostic.SyntaxError](err)
	require.True(t, ok, "expected *diagnostic.SyntaxError, got %v", err)

	return se.Code
}

func casePermutations(prefix string) []string {
	out := []string{""}
	for _, c := range []byte(prefix) {
		var next []string
		for _, head := range out {
			next = append(next, head+string(c), head+strings.ToUpper(string(c)))
		}
		out = next
	}

	return out
}
