package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-Bython/internal/bython"
	"go-Bython/internal/bython/tokentest"
)

type (
	streamInput struct {
		name string
		src  string
	}
)

const (
	readBlockSize = 64 << 10
)

func TestStreamMatchesTranslateForEdgeCases(t *testing.T) {
	for _, input := range streamInputs() {
		t.Run(input.name, func(t *testing.T) {
			//given
			r := &tokentest.OneByteReader{R: strings.NewReader(input.src)}
			want, wantErr := bython.Translate(input.src, 2)

			//when
			var out strings.Builder
			err := bython.TranslateStream(r, &out, 2)

			//then
			assertSameResult(t, want, wantErr, out.String(), err)
		})
	}
}

func TestStreamMatchesTranslateAtEveryBlockOffset(t *testing.T) {
	for _, input := range streamInputs() {
		t.Run(input.name, func(t *testing.T) {
			for offset := range len(input.src) + 1 {
				//given
				filler := "#" + strings.Repeat("x", readBlockSize-offset-2) + "\n"
				src := filler + input.src
				want, wantErr := bython.Translate(src, 2)

				//when
				var out strings.Builder
				err := bython.TranslateStream(strings.NewReader(src), &out, 2)

				//then
				assertSameResult(t, want, wantErr, out.String(), err)
			}
		})
	}
}

func TestStreamMatchesTranslateWithOperatorSplitInFStringField(t *testing.T) {
	//given
	src := "# " + strings.Repeat("a", readBlockSize-16) + "\nx = f'''{(a\n!= b)}'''\n"
	want, wantErr := bython.Translate(src, 2)
	require.NoError(t, wantErr)

	//when
	var out strings.Builder
	err := bython.TranslateStream(strings.NewReader(src), &out, 2)

	//then
	assertSameResult(t, want, wantErr, out.String(), err)
}

func TestProcessFileTranslatesInPlace(t *testing.T) {
	//given
	dir := t.TempDir()
	path := filepath.Join(dir, "in.py")
	require.NoError(t, os.WriteFile(path, []byte("if x {\n    y();\n}"), 0o644))
	p := NewPythonPreprocessor(2)

	//when
	_, err := p.ProcessFile(path, path)

	//then
	require.NoError(t, err)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "if x:\n  y()\n", string(content))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestProcessFileErrorInLaterChunkCreatesNoOutput(t *testing.T) {
	//given
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "bad.py")
	outputPath := filepath.Join(dir, "out.py")
	input := strings.Repeat("x = 1;\n", 20000) + "if x {\n    pass;"
	require.NoError(t, os.WriteFile(inputPath, []byte(input), 0o644))
	p := NewPythonPreprocessor(2)

	//when
	_, err := p.ProcessFile(inputPath, outputPath)

	//then
	require.Error(t, err)
	assert.Equal(t, inputPath+":20001:6: '{' was never closed (opens the 'if' block)", err.Error())
	assert.NoFileExists(t, outputPath)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "bad.py", entries[0].Name())
}

func TestProcessFileReplacesExistingOutput(t *testing.T) {
	//given
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "in.py")
	outputPath := filepath.Join(dir, "out.py")
	require.NoError(t, os.WriteFile(inputPath, []byte("if x {\n    y();\n}"), 0o644))
	require.NoError(t, os.WriteFile(outputPath, []byte("stale"), 0o644))
	p := NewPythonPreprocessor(2)

	//when
	timings, err := p.ProcessFile(inputPath, outputPath)

	//then
	require.NoError(t, err)
	assert.Positive(t, timings.Total)
	assert.GreaterOrEqual(t, timings.Total, timings.Open+timings.Read+timings.Translate+timings.Write+timings.Finalise)
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "if x:\n  y()\n", string(content))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestProcessReaderKeepsFlushedOutputOnLaterError(t *testing.T) {
	//given
	input := strings.Repeat("x = 1;\n", 20000) + "if x {\n    pass;"
	complete := strings.Repeat("x = 1\n", 20000)
	p := NewPythonPreprocessor(2)

	//when
	var out strings.Builder
	err := p.ProcessReader(strings.NewReader(input), &out)

	//then
	require.Error(t, err)
	assert.Equal(t, "20001:6: '{' was never closed (opens the 'if' block)", err.Error())
	assert.NotZero(t, out.Len())
	assert.True(t, strings.HasPrefix(complete, out.String()))
}

func streamInputs() []streamInput {
	var inputs []streamInput
	for _, c := range edgeCases {
		inputs = append(inputs, streamInput{"edge/" + c.name, c.input})
	}

	for _, c := range errorCases {
		inputs = append(inputs, streamInput{"error/" + c.name, c.input})
	}

	return inputs
}

func assertSameResult(t *testing.T, want string, wantErr error, got string, err error) {
	t.Helper()

	if wantErr != nil {
		require.Error(t, err)
		assert.Equal(t, wantErr.Error(), err.Error())
		return
	}

	require.NoError(t, err)
	assert.Equal(t, want, got)
}
