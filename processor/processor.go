package processor

import "io"

type Processor interface {
	ProcessReader(reader io.Reader, writer io.Writer) error
	ProcessString(input string) string
	ProcessFile(inputPath, outputPath string) error
	IndentSize() int
}
