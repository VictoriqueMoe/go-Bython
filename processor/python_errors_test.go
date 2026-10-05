package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errorCases = []struct {
	name    string
	input   string
	wantErr string
}{
	{"unbalanced_open_brace", "if x {\n    pass;", "1:6: '{' was never closed (opens the 'if' block)"},
	{"unbalanced_close_brace", "x = 1;\n}\ny = 2;", "2:1: unmatched '}': no block or bracket is open"},
	{"unbalanced_bracket_literal", "d = {\n  \"a\": 1\nif x {\n    pass;\n}", "1:5: '{' was never closed"},
	{"unterminated_string", "x = \"abc;\nif x {\n    pass;\n}", "1:5: unterminated string literal"},
	{"unterminated_triple_quoted_string", "s = \"\"\"abc\nif x {\n    pass;\n}", "1:5: unterminated triple-quoted string literal"},
	{"mismatched_brackets", "x = (1, 2];", "1:10: closing ']' does not match '(' opened at 1:5"},
	{"mismatched_in_block", "if a {\n  f(1, 2 }\n", "2:10: closing '}' does not match '(' opened at 2:4"},
	{"unmatched_paren", "x = 1)", "1:6: unmatched ')'"},
	{"block_after_simple", "foo() {\n}", "1:7: '{' cannot open a block here: only compound statements such as 'if', 'for', 'while', 'def' or 'class' take a block"},
	{"missing_block_brace", "if x\n  y\n", "1:5: expected '{' to open the 'if' block"},
	{"missing_block_brace_after_multiline_string", "if \"\"\"a\nb\"\"\"\nx = 1", "2:5: expected '{' to open the 'if' block"},
	{"python_colon_style", "if x:\n  y = 1\n", "1:5: expected '{' to open the 'if' block (Bython blocks use '{', not ':')"},
	{"orphan_else", "else {\n}", "1:1: 'else' without a matching 'if', 'for', 'while' or 'try'"},
	{"else_after_else", "if a {} else {} else {}", "1:17: 'else' cannot follow 'else' at 1:9"},
	{"else_if", "if a {\n} else if b {\n}", "2:3: 'else' must be followed directly by '{'; use 'elif' for 'else if'"},
	{"try_without_handler", "try {\n}\nx = 1", "1:1: 'try' block needs an 'except' or 'finally' block"},
	{"mixed_except", "try {\n} except A {\n} except* B {\n}", "3:3: cannot mix 'except' and 'except*'"},
	{"decorator_target", "@d\nx = 1;", "2:1: a decorator must be followed by 'def', 'class' or 'async def'"},
	{"async_without_target", "async x = 1;", "1:1: 'async' must be followed by 'def', 'for' or 'with'"},
	{"case_outside_match", "case 1 {\n}", "1:1: 'case' block outside 'match'"},
	{"not_case_in_match", "match x {\n    y = 1;\n}", "2:5: only 'case' blocks may appear inside 'match'"},
	{"match_without_case", "match x {\n}", "1:1: 'match' block needs at least one 'case' block"},
	{"juxtaposed_operands", "print \"x\";", "1:7: missing operator, ',' or ';' between expressions"},
	{"fstring_single_brace", "x = f\"}\";", "1:7: f-string: single '}' is not allowed"},
	{"fstring_unclosed_field", "x = f\"{a\";", "1:7: f-string: expecting '}'"},
	{"invalid_character", "x = $;", "1:5: invalid character '$' (U+0024)"},
	{"bad_continuation", "a = 1 \\ + 2", "1:7: unexpected character after line continuation character"},
}

func TestErrorCases(t *testing.T) {
	for _, c := range errorCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			p := NewPythonPreprocessor(2)

			//when
			result, err := p.ProcessString(c.input)

			//then
			require.Error(t, err)
			assert.Equal(t, c.wantErr, err.Error())
			assert.Equal(t, "", result)
		})
	}
}

func TestErrorCasesProcessReader(t *testing.T) {
	for _, c := range errorCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			p := NewPythonPreprocessor(2)
			var builder strings.Builder

			//when
			err := p.ProcessReader(strings.NewReader(c.input), &builder)

			//then
			require.Error(t, err)
			assert.Equal(t, c.wantErr, err.Error())
			assert.Equal(t, 0, builder.Len())
		})
	}
}

func TestTooManyNestedBlocks(t *testing.T) {
	//given
	input := strings.Repeat("if x {\n", 101)
	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)

	//then
	require.Error(t, err)
	assert.Equal(t, "101:6: too many nested blocks (limit 100)", err.Error())
	assert.Equal(t, "", result)
}

func TestProcessFileSyntaxErrorCreatesNoOutput(t *testing.T) {
	//given
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "bad.py")
	outputPath := filepath.Join(dir, "out.py")
	require.NoError(t, os.WriteFile(inputPath, []byte("if x {\n    pass;"), 0o644))
	p := NewPythonPreprocessor(2)

	//when
	err := p.ProcessFile(inputPath, outputPath)

	//then
	require.Error(t, err)
	assert.Equal(t, inputPath+":1:6: '{' was never closed (opens the 'if' block)", err.Error())
	assert.NoFileExists(t, outputPath)
}
