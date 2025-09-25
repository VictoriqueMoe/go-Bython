package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go-Bython/processor"
)

func main() {
	var (
		inputFile   = flag.String("i", "", "Input file path (required for single file mode)")
		outputFile  = flag.String("o", "", "Output file path (required for single file mode)")
		inputDir    = flag.String("d", "", "Input directory for batch processing")
		outputDir   = flag.String("od", "", "Output directory for batch processing")
		filePattern = flag.String("pattern", "*.py", "File pattern to match (e.g., '*.py', '*.pybrace')")
		workers     = flag.Int("workers", 4, "Number of concurrent workers for batch processing")
		indentSize  = flag.Int("indent", 2, "Number of spaces for indentation")
	)
	flag.Parse()

	p := processor.NewPythonPreprocessor(*indentSize)

	if *inputDir != "" {
		if *outputDir == "" {
			log.Fatal("output directory (-od) is required when using input directory (-d)")
		}

		start := time.Now()
		fp := processor.NewFolderProcessor(*indentSize, *filePattern, *workers)
		if err := fp.ProcessFolder(*inputDir, *outputDir); err != nil {
			log.Fatal(err)
			return
		}
		fmt.Printf("Successfully processed folder: %s -> %s in %v\n", *inputDir, *outputDir, time.Since(start))
		return
	}

	if *inputFile == "" || *outputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	start := time.Now()
	if err := p.ProcessFile(*inputFile, *outputFile); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Successfully processed: %s -> %s in %v\n", *inputFile, *outputFile, time.Since(start))
}

func init() {
	flag.Usage = func() {
		fmt.Println(fmt.Sprintf("Usage: %s [options]", os.Args[0]))
		fmt.Println(fmt.Sprintf("\nA preprocessor that converts brace-style Python to indented Python."))
		fmt.Println(fmt.Sprintf("\nOptions:"))
		flag.PrintDefaults()
		fmt.Println(fmt.Sprintf("\nExamples:"))
		fmt.Println(fmt.Sprintf("  Single file:"))
		fmt.Println(fmt.Sprintf("    go-bython -i input.py -o output.py"))
		fmt.Println(fmt.Sprintf("    go-bython -i input.py -o output.py -indent 4"))
		fmt.Println(fmt.Sprintf("\n  Batch processing:"))
		fmt.Println(fmt.Sprintf("    go-bython -d ./src -od ./output"))
		fmt.Println(fmt.Sprintf("    go-bython -d ./src -od ./output -pattern '*.pybrace' -workers 8"))
	}
}
