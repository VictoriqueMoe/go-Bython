package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFolderProcessor(t *testing.T) {
	//given
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFiles := map[string]string{
		"test1.py": `if x > 0 {
    print("positive");
}`,
		"test2.py": `def foo() {
    return 42;
}`,
		"subdir/test3.py": `while True {
    break;
}`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(inputDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	fp := NewFolderProcessor(2, "*.py", 2)

	expectedFiles := map[string]string{
		"test1.py": `if x > 0:
  print("positive")
`,
		"test2.py": `def foo():
  return 42
`,
		filepath.Join("subdir", "test3.py"): `while True:
  break
`,
	}

	//when
	err := fp.ProcessFolder(inputDir, outputDir)

	//then
	assert.NoError(t, err)

	for path, expected := range expectedFiles {
		fullPath := filepath.Join(outputDir, path)
		content, err := os.ReadFile(fullPath)
		assert.NoError(t, err, "Failed to read output file %s", path)
		assert.Equal(t, expected, string(content), "File %s content mismatch", path)
	}
}

func TestFolderProcessorWithPattern(t *testing.T) {
	//given
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"test.pybrace": `if x { print("test"); }`,
		"ignore.py":    `if x { print("ignore"); }`,
	}

	for name, content := range files {
		path := filepath.Join(inputDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	fp := NewFolderProcessor(4, "*.pybrace", 2)

	//when
	err := fp.ProcessFolder(inputDir, outputDir)

	//then
	assert.NoError(t, err)

	processedFile := filepath.Join(outputDir, "test.py")
	assert.FileExists(t, processedFile, "Expected test.py to be created")

	ignoredFile := filepath.Join(outputDir, "ignore.py")
	assert.NoFileExists(t, ignoredFile, "Expected ignore.py to NOT be created")
}
