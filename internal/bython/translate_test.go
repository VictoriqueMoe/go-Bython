package bython

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-Bython/internal/bython/emitter"
	"go-Bython/internal/bython/lexer"
	"go-Bython/internal/bython/parser"
	"go-Bython/internal/bython/tokentest"
)

var (
	chunkedCases = []struct {
		name     string
		input    string
		expected string
	}{
		{
			"boundary_lookahead_reaches_elif_past_comment_and_blank",
			"if a {\n    x();\n}\n# note\n\nelif b {\n    y();\n}\nz();",
			"if a:\n  x()\n# note\n\nelif b:\n  y()\nz()\n",
		},
		{
			"decorator_and_def_split_over_lines",
			"@dec\n\n# c\n@other(\n    1\n)\ndef f() {\n    pass;\n}\nx = 1;",
			"@dec\n\n# c\n@other(\n    1\n)\ndef f():\n  pass\nx = 1\n",
		},
		{
			"allman_brace_after_comment_line",
			"if x\n# c\n{\n    a();\n}\ny();",
			"if x:\n  # c\n  a()\ny()\n",
		},
		{
			"comment_run_after_simple_statement_split_per_line",
			"x = 1;\n# a\n\n# b\ny = 2;\n# c\n",
			"x = 1\n# a\n\n# b\ny = 2\n# c\n",
		},
		{
			"statements_split_into_chunks",
			"a = 1;\n\n\nb = {\n    1: 2\n};\nclass C {\n    pass;\n}\n\n",
			"a = 1\n\n\nb = {\n    1: 2\n}\nclass C:\n  pass\n\n",
		},
	}

	laterChunkErrors = []struct {
		name    string
		input   string
		wantErr string
	}{
		{"parse_error", strings.Repeat("a = 1;\n", 20000) + "c = (1, 2];", "20001:10: closing ']' does not match '(' opened at 20001:5"},
		{"lex_error", strings.Repeat("a = 1;\n", 20000) + "x = $;", "20001:5: invalid character '$' (U+0024)"},
		{"unclosed_block", strings.Repeat("a = 1;\n", 20000) + "if x {\n    pass;", "20001:6: '{' was never closed (opens the 'if' block)"},
	}
)

func TestChunkedTranslation(t *testing.T) {
	for _, c := range chunkedCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			want, err := wholeTranslate(c.input, 2)
			require.NoError(t, err)
			require.Equal(t, c.expected, want)

			//when
			got, err := Translate(c.input, 2)
			var streamed strings.Builder
			streamErr := TranslateStream(&tokentest.OneByteReader{R: strings.NewReader(c.input)}, &streamed, 2)

			//then
			require.NoError(t, err)
			require.NoError(t, streamErr)
			assert.Equal(t, c.expected, got)
			assert.Equal(t, c.expected, streamed.String())
		})
	}
}

func TestTripleQuotedStringAcrossBlockBoundary(t *testing.T) {
	//given
	body := strings.Repeat("ab\n", 30000)
	input := "s = '''" + body + "''';\nx = 1;"
	expected := "s = '''" + body + "'''\nx = 1\n"
	r := &tokentest.OneByteReader{R: strings.NewReader(input)}

	//when
	var out strings.Builder
	err := TranslateStream(r, &out, 2)

	//then
	require.NoError(t, err)
	assert.Equal(t, expected, out.String())
}

func TestErrorInLaterChunk(t *testing.T) {
	for _, c := range laterChunkErrors {
		t.Run(c.name, func(t *testing.T) {
			//given
			r := &tokentest.OneByteReader{R: strings.NewReader(c.input)}

			//when
			result, err := Translate(c.input, 2)
			var streamed strings.Builder
			streamErr := TranslateStream(r, &streamed, 2)

			//then
			require.Error(t, err)
			require.Error(t, streamErr)
			assert.Equal(t, c.wantErr, err.Error())
			assert.Equal(t, c.wantErr, streamErr.Error())
			assert.Equal(t, "", result)
		})
	}
}

func FuzzChunkedTranslate(f *testing.F) {
	addSeeds(f)

	f.Fuzz(func(t *testing.T, src string) {
		want, wantErr := wholeTranslate(src, 2)
		got, err := Translate(src, 2)

		if wantErr != nil {
			if err == nil {
				t.Fatalf("whole translation failed with %v but chunked succeeded\ninput: %q", wantErr, src)
			}
			return
		}

		if err != nil {
			t.Fatalf("chunked translation failed with %v but whole succeeded\ninput: %q", err, src)
		}

		if got != want {
			t.Fatalf("chunked output differs\ninput: %q\nwhole: %q\nchunked: %q", src, want, got)
		}
	})
}

func wholeTranslate(src string, indentSize int) (string, error) {
	s, err := lexer.Lex(src)
	if err != nil {
		return "", err
	}

	m, err := parser.Parse(s)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	w := bufio.NewWriter(&b)

	var em emitter.Emitter
	if err := em.Emit(w, m, indentSize); err != nil {
		return "", err
	}

	if err := w.Flush(); err != nil {
		return "", err
	}

	return b.String(), nil
}
