package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-Bython/internal/bython/token"
	"go-Bython/internal/bython/tokentest"
)

var (
	chunkCases = []struct {
		name   string
		src    string
		chunks [][]string
	}{
		{"one_chunk_per_statement", "a = 1\nb = 2\n", [][]string{
			{"a", "=", "1", "\n"},
			{"b", "=", "2", "\n"},
		}},
		{"lookahead_keeps_elif_after_comment_and_blank", "if a {\n}\n# c\n\nelif b {\n}\nx\n", [][]string{
			{"if", "a", "{", "\n", "}", "\n", "# c", "\n", "\n", "elif", "b", "{", "\n", "}", "\n"},
			{"x", "\n"},
		}},
		{"trivia_after_simple_statement_splits_per_line", "a\n# c\n\nb\n", [][]string{
			{"a", "\n"},
			{"# c", "\n"},
			{"\n", "b", "\n"},
		}},
		{"lookahead_trivia_after_block_moves_to_next_chunk", "if a {\n}\n# c\n\nb\n", [][]string{
			{"if", "a", "{", "\n", "}", "\n"},
			{"# c", "\n", "\n", "b", "\n"},
		}},
		{"elif_after_comment_following_simple_statement_stays", "a\n# c\nelif b {\n}\n", [][]string{
			{"a", "\n"},
			{"# c", "\n", "elif", "b", "{", "\n", "}", "\n"},
		}},
		{"close_and_else_line_waits_for_allman_brace", "if a {\n} else\n# c\n{\n}\nx\n", [][]string{
			{"if", "a", "{", "\n", "}", "else", "\n", "# c", "\n", "{", "\n", "}", "\n"},
			{"x", "\n"},
		}},
		{"decorator_and_def_split_over_lines", "@dec\n\n# c\n@other(\n    1\n)\ndef f() {\n}\nx\n", [][]string{
			{"@", "dec", "\n", "\n", "# c", "\n", "@", "other", "(", "\n", "1", "\n", ")", "\n", "def", "f", "(", ")", "{", "\n", "}", "\n"},
			{"x", "\n"},
		}},
		{"allman_brace_after_comment_line", "if x\n# c\n{\n}\ny\n", [][]string{
			{"if", "x", "\n", "# c", "\n", "{", "\n", "}", "\n"},
			{"y", "\n"},
		}},
		{"try_chain_stays_together", "try {\n}\nexcept E {\n}\nfinally {\n}\nz\n", [][]string{
			{"try", "{", "\n", "}", "\n", "except", "E", "{", "\n", "}", "\n", "finally", "{", "\n", "}", "\n"},
			{"z", "\n"},
		}},
		{"open_bracket_spans_newlines", "d = {\n  1: 2\n}\ne = 3\n", [][]string{
			{"d", "=", "{", "\n", "1", ":", "2", "\n", "}", "\n"},
			{"e", "=", "3", "\n"},
		}},
		{"negative_depth_never_splits", "}\na\nb\n", [][]string{
			{"}", "\n", "a", "\n", "b", "\n"},
		}},
		{"trailing_whitespace_line_stays_in_last_chunk", "a\n   ", [][]string{
			{"a", "\n", ""},
		}},
	}
)

func TestStreamChunkBoundaries(t *testing.T) {
	for _, c := range chunkCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			r := &tokentest.OneByteReader{R: strings.NewReader(c.src)}

			//when
			chunks := lexChunks(t, r)

			//then
			assert.Equal(t, c.chunks, chunks)
		})
	}
}

func TestStreamRebasesLaterChunks(t *testing.T) {
	//given
	var s token.Source
	var st Stream
	st.Reset(&s, &tokentest.OneByteReader{R: strings.NewReader("a = 1\n\n  b = 2\n")})

	//when
	done, err := st.Next()
	require.NoError(t, err)
	require.False(t, done)

	done, err = st.Next()

	//then
	require.NoError(t, err)
	require.True(t, done)
	assert.Equal(t, "\n  b = 2\n", s.Text)

	blank := s.Tokens[0]
	assert.Equal(t, token.KindNewline, blank.Kind)
	assert.Equal(t, int32(2), blank.Line)
	assert.Equal(t, int32(0), s.LineOf(blank).Start)

	b := s.Tokens[1]
	assert.Equal(t, "b", s.Text[b.Start:b.End])
	assert.Equal(t, int32(3), b.Line)
	assert.Equal(t, int32(1), s.LineOf(b).Start)
	assert.Equal(t, int32(2), s.LineOf(b).Indent)
}

func TestStreamTripleStringAcrossBlocks(t *testing.T) {
	//given
	literal := "'''" + strings.Repeat("ab\n", 30000) + "'''"
	src := "s = " + literal + "\nx\n"
	r := &tokentest.OneByteReader{R: strings.NewReader(src)}

	//when
	chunks := lexChunks(t, r)

	//then
	assert.Equal(t, [][]string{{"s", "=", literal, "\n"}, {"x", "\n"}}, chunks)
}

func TestStreamMatchesLexOnLongLine(t *testing.T) {
	//given
	src := "x = '" + strings.Repeat("a", 3*minBlock) + "' # c\ny = 1\n"
	whole, err := Lex(src)
	require.NoError(t, err)

	//when
	chunks := lexChunks(t, &tokentest.OneByteReader{R: strings.NewReader(src)})

	//then
	var flat []string
	for _, chunk := range chunks {
		flat = append(flat, chunk...)
	}
	assert.Equal(t, texts(whole), flat)
}

func TestStreamOperatorPrefixAtWindowEndInFStringField(t *testing.T) {
	//given
	src := "# " + strings.Repeat("a", minBlock-16) + "\nx = f'''{(a\n!= b)}'''\n"
	whole, err := Lex(src)
	require.NoError(t, err)

	//when
	chunks := lexChunks(t, &tokentest.OneByteReader{R: strings.NewReader(src)})

	//then
	var flat []string
	for _, chunk := range chunks {
		flat = append(flat, chunk...)
	}
	assert.Equal(t, texts(whole), flat)
}

func TestStreamCommentRunAfterSimpleStatementKeepsWindowSmall(t *testing.T) {
	//given
	src := "x = 1\n" + strings.Repeat("# c\n\n", 1<<20) + "y = 2\n"
	var s token.Source
	var st Stream
	st.Reset(&s, strings.NewReader(src))

	//when
	maxText, maxTokens := 0, 0
	for {
		done, err := st.Next()
		require.NoError(t, err)

		maxText = max(maxText, len(s.Text))
		maxTokens = max(maxTokens, len(s.Tokens))

		if done {
			break
		}
	}

	//then
	assert.LessOrEqual(t, maxText, 4*minBlock)
	assert.LessOrEqual(t, maxTokens, 8)
}

func TestStreamErrorsMatchLex(t *testing.T) {
	for _, c := range lexErrorCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			var s token.Source
			var st Stream
			st.Reset(&s, &tokentest.OneByteReader{R: strings.NewReader(c.src)})
			_, want := Lex(c.src)

			//when
			_, err := st.Next()

			//then
			require.Error(t, err)
			assert.Equal(t, c.code, errorCode(t, err))
			assert.Equal(t, want.Error(), err.Error())
		})
	}
}

func lexChunks(t *testing.T, r *tokentest.OneByteReader) [][]string {
	t.Helper()

	var s token.Source
	var st Stream
	st.Reset(&s, r)

	var out [][]string
	for {
		done, err := st.Next()
		require.NoError(t, err)

		require.NotEmpty(t, s.Tokens)
		require.Equal(t, token.KindEOF, s.Tokens[len(s.Tokens)-1].Kind)
		out = append(out, texts(&s))

		if done {
			return out
		}
	}
}

func texts(s *token.Source) []string {
	var out []string
	for _, tok := range s.Tokens {
		if tok.Kind == token.KindEOF {
			continue
		}
		out = append(out, s.Text[tok.Start:tok.End])
	}

	return out
}
