package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FolderProcessor struct {
	indentSize  int
	filePattern string
	workers     int
}

func NewFolderProcessor(indentSize int, filePattern string, workers int) *FolderProcessor {
	if workers <= 0 {
		workers = 4
	}
	if indentSize <= 0 {
		indentSize = 4
	}
	return &FolderProcessor{
		indentSize:  indentSize,
		filePattern: filePattern,
		workers:     workers,
	}
}

func (f *FolderProcessor) ProcessFolder(inputDir, outputDir string) error {
	files, err := f.discoverFiles(inputDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files matching pattern '%s' found in %s", f.filePattern, inputDir)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	return f.processFilesConcurrently(inputDir, outputDir, files)
}

func (f *FolderProcessor) discoverFiles(rootDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			matched, err := filepath.Match(f.filePattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if matched {
				files = append(files, path)
			}
		}

		return nil
	})

	return files, err
}

func (f *FolderProcessor) processFilesConcurrently(inputDir, outputDir string, files []string) error {
	type result struct {
		file string
		err  error
	}

	jobs := make(chan string, len(files))
	results := make(chan result, len(files))

	var wg sync.WaitGroup

	for i := 0; i < f.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localProcessor := NewPythonPreprocessor(f.indentSize)
			for file := range jobs {
				relPath, err := filepath.Rel(inputDir, file)
				if err != nil {
					results <- result{file, err}
					continue
				}

				outputPath := filepath.Join(outputDir, relPath)
				outputFileDir := filepath.Dir(outputPath)

				if err := os.MkdirAll(outputFileDir, 0755); err != nil {
					results <- result{file, fmt.Errorf("failed to create directory %s: %v", outputFileDir, err)}
					continue
				}

				outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".py"

				if err := localProcessor.ProcessFile(file, outputPath); err != nil {
					results <- result{file, err}
					continue
				}

				results <- result{file, nil}
			}
		}()
	}

	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var errors []string
	processed := 0

	for res := range results {
		if res.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", res.file, res.err))
		} else {
			processed++
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("processed %d files with %d errors:\n%s", processed, len(errors), strings.Join(errors, "\n"))
	}

	return nil
}
