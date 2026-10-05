package processor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go-Bython/internal/bython/diagnostic"
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

func (f *FolderProcessor) ProcessFolder(inputDir, outputDir string) (FolderSummary, error) {
	wall := startStopwatch()

	files, err := f.discoverFiles(inputDir)
	if err != nil {
		return FolderSummary{}, err
	}

	if len(files) == 0 {
		return FolderSummary{}, fmt.Errorf("no files matching pattern '%s' found in %s", f.filePattern, inputDir)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return FolderSummary{}, fmt.Errorf("failed to create output directory: %v", err)
	}

	summary, err := f.processFilesConcurrently(inputDir, outputDir, files)
	summary.Wall = wall.elapsed()

	return summary, err
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

func (f *FolderProcessor) processFilesConcurrently(inputDir, outputDir string, files []string) (FolderSummary, error) {
	type result struct {
		file    string
		timings Timings
		err     error
	}

	jobs := make(chan string, len(files))
	results := make(chan result, len(files))

	var wg sync.WaitGroup

	for i := 0; i < f.workers; i++ {
		wg.Go(func() {
			localProcessor := NewPythonPreprocessor(f.indentSize)
			for file := range jobs {
				relPath, err := filepath.Rel(inputDir, file)
				if err != nil {
					results <- result{file: file, err: err}
					continue
				}

				outputPath := filepath.Join(outputDir, relPath)
				outputFileDir := filepath.Dir(outputPath)

				if err := os.MkdirAll(outputFileDir, 0755); err != nil {
					results <- result{file: file, err: fmt.Errorf("failed to create directory %s: %v", outputFileDir, err)}
					continue
				}

				outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".py"

				timings, err := localProcessor.ProcessFile(file, outputPath)
				results <- result{file: file, timings: timings, err: err}
			}
		})
	}

	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var failures []string
	var summary FolderSummary

	for res := range results {
		if res.err == nil {
			summary.Files++
			summary.Timings.Add(res.timings)
			continue
		}

		if _, ok := errors.AsType[*diagnostic.SyntaxError](res.err); ok {
			failures = append(failures, res.err.Error())
		} else {
			failures = append(failures, fmt.Sprintf("%s: %v", res.file, res.err))
		}
	}

	if len(failures) > 0 {
		return summary, fmt.Errorf("processed %d files with %d errors:\n%s", summary.Files, len(failures), strings.Join(failures, "\n"))
	}

	return summary, nil
}
