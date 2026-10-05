package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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

		fp := processor.NewFolderProcessor(*indentSize, *filePattern, *workers)
		summary, err := fp.ProcessFolder(*inputDir, *outputDir)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Successfully processed folder: %s -> %s\n", *inputDir, *outputDir)
		fmt.Printf("Summed across %d files:\n", summary.Files)
		printSteps(summary.Timings.Steps())
		fmt.Printf("  %-10s %v\n", "wall", summary.Wall)
		return
	}

	if *inputFile == "" || *outputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	timings, err := p.ProcessFile(*inputFile, *outputFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Successfully processed: %s -> %s\n", *inputFile, *outputFile)
	printSteps(timings.Steps())
}

func printSteps(steps []processor.TimingStep) {
	for _, step := range steps {
		fmt.Printf("  %-10s %v\n", step.Name, step.Duration)
	}
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
