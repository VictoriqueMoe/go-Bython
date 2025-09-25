package processor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type PythonPreprocessor struct {
	indentSize       int
	indentChar       string
	indentLevel      int
	structuralBlocks int
	dictDepth        int
	dictBaseIndent   int
}

func NewPythonPreprocessor(indentSize int) Processor {
	return &PythonPreprocessor{
		indentSize:       indentSize,
		indentChar:       strings.Repeat(" ", indentSize),
		indentLevel:      0,
		structuralBlocks: 0,
		dictDepth:        0,
	}
}

var controlKeywords = []string{
	"if ", "elif ", "else", "while ", "for ", "def ", "class ", "try", "except", "finally", "with ",
}

func (p *PythonPreprocessor) processLine(line string) []string {
	trimmed := strings.TrimSpace(p.handleSingleLineBraces(line))
	if trimmed == "" {
		return []string{""}
	}

	if strings.HasPrefix(trimmed, "#") {
		return []string{p.indent() + trimmed}
	}

	if strings.HasPrefix(trimmed, "}") {
		if p.dictDepth > 0 {
			p.dictDepth--
			leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
			relativeIndent := leadingSpaces - p.dictBaseIndent
			indentLevels := relativeIndent / p.indentSize
			if relativeIndent > 0 && indentLevels == 0 {
				indentLevels = 1
			}
			normalizedIndent := strings.Repeat(p.indentChar, indentLevels)
			processedLine := normalizedIndent + strings.TrimSuffix(trimmed, ";")
			if p.dictDepth == 0 {
				p.dictBaseIndent = 0
			}
			return []string{processedLine}
		}

		if p.structuralBlocks > 0 {
			p.indentLevel--
			p.structuralBlocks--

			remaining := strings.TrimSpace(trimmed[1:])

			if strings.HasPrefix(remaining, "else") || strings.HasPrefix(remaining, "elif") {
				return p.processLine(remaining)
			} else if remaining != "" {
				return p.processLine(remaining)
			}
			return []string{}
		}

		processedLine := strings.TrimSuffix(trimmed, ";")
		return []string{p.indent() + processedLine}
	}

	if p.dictDepth > 0 {
		leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
		relativeIndent := leadingSpaces - p.dictBaseIndent
		indentLevels := relativeIndent / p.indentSize
		if relativeIndent > 0 && indentLevels == 0 {
			indentLevels = 1
		}
		normalizedIndent := strings.Repeat(p.indentChar, indentLevels)
		processedLine := normalizedIndent + strings.TrimSuffix(trimmed, ";")
		openBraces := strings.Count(processedLine, "{")
		closeBraces := strings.Count(processedLine, "}")
		p.dictDepth += openBraces - closeBraces
		return []string{processedLine}
	}

	dictBraceIndex := p.findDictionaryBrace(trimmed)
	if dictBraceIndex != -1 {
		p.dictBaseIndent = len(line) - len(strings.TrimLeft(line, " \t"))
		p.dictDepth = 1
		openBraces := strings.Count(trimmed, "{")
		closeBraces := strings.Count(trimmed, "}")
		p.dictDepth = openBraces - closeBraces
		processedLine := p.indent() + strings.TrimSuffix(trimmed, ";")
		return []string{processedLine}
	}

	needsColon := false
	openBraceIndex := p.findStructuralBrace(trimmed)

	if openBraceIndex != -1 {
		beforeBrace := strings.TrimSpace(trimmed[:openBraceIndex])
		afterBrace := strings.TrimSpace(trimmed[openBraceIndex+1:])

		if !p.isControlStatement(beforeBrace) {
			processedLine := strings.TrimSuffix(trimmed, ";")
			return []string{p.indent() + processedLine}
		}

		if p.isControlStatement(beforeBrace) {
			needsColon = true
		}

		processedLine := beforeBrace
		if needsColon && !strings.HasSuffix(processedLine, ":") {
			processedLine += ":"
		}

		result := []string{p.indent() + processedLine}

		p.indentLevel++
		p.structuralBlocks++

		if afterBrace != "" && afterBrace != "}" {
			if strings.HasSuffix(afterBrace, "}") {
				content := strings.TrimSpace(afterBrace[:len(afterBrace)-1])
				if content != "" {
					content = strings.TrimSuffix(content, ";")
					result = append(result, p.indent()+content)
				}
				p.indentLevel--
				p.structuralBlocks--
			} else {
				afterBrace = strings.TrimSuffix(afterBrace, ";")
				result = append(result, p.indent()+afterBrace)
			}
		}

		return result
	}

	processedLine := trimmed
	processedLine = strings.TrimSuffix(processedLine, ";")

	return []string{p.indent() + processedLine}
}

func (p *PythonPreprocessor) handleSingleLineBraces(line string) string {
	re := regexp.MustCompile(`lambda\s+([^{]+?)\s*\{([^}]*)\}`)
	return re.ReplaceAllString(line, `lambda $1: $2`)
}

func (p *PythonPreprocessor) isControlStatement(line string) bool {
	for _, keyword := range controlKeywords {
		if strings.HasPrefix(line, keyword) || line == strings.TrimSpace(keyword) {
			return true
		}
	}

	if strings.Contains(line, "__main__") {
		return true
	}

	return false
}

func (p *PythonPreprocessor) indent() string {
	return strings.Repeat(p.indentChar, p.indentLevel)
}

func (p *PythonPreprocessor) findStructuralBrace(line string) int {
	inString := false
	stringChar := rune(0)
	inFString := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if i > 0 && (line[i-1] == 'f' || line[i-1] == 'F') && (ch == '"' || ch == '\'') {
			inFString = true
		}

		if (ch == '"' || ch == '\'') && (i == 0 || line[i-1] != '\\') {
			if !inString {
				inString = true
				stringChar = rune(ch)
			} else if rune(ch) == stringChar {
				inString = false
				stringChar = 0
				inFString = false
			}
		}

		if ch == '{' && !inString {
			if p.isDictionaryBrace(line, i) {
				continue
			}
			return i
		}

		if ch == '{' && inFString {
			depth := 1
			for j := i + 1; j < len(line); j++ {
				if line[j] == '{' {
					depth++
				} else if line[j] == '}' {
					depth--
					if depth == 0 {
						i = j
						break
					}
				}
			}
		}
	}
	return -1
}

func (p *PythonPreprocessor) findDictionaryBrace(line string) int {
	inString := false
	stringChar := rune(0)
	inFString := false
	dictBraceIndex := -1
	depth := 0

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if i > 0 && (line[i-1] == 'f' || line[i-1] == 'F') && (ch == '"' || ch == '\'') {
			inFString = true
		}

		if (ch == '"' || ch == '\'') && (i == 0 || line[i-1] != '\\') {
			if !inString {
				inString = true
				stringChar = rune(ch)
			} else if rune(ch) == stringChar {
				inString = false
				stringChar = 0
				inFString = false
			}
		}

		if ch == '{' && inFString {
			depth := 1
			for j := i + 1; j < len(line); j++ {
				if line[j] == '{' {
					depth++
				} else if line[j] == '}' {
					depth--
					if depth == 0 {
						i = j
						break
					}
				}
			}
			continue
		}

		if ch == '{' && !inString {
			before := strings.TrimSpace(line[:i])
			if strings.HasSuffix(before, ")") {
				continue
			}
			if strings.HasSuffix(before, "=") || strings.HasSuffix(before, ":") || strings.HasSuffix(before, "(") || strings.HasSuffix(before, "[") || strings.HasSuffix(before, ",") || strings.HasSuffix(before, "return") {
				if dictBraceIndex == -1 {
					dictBraceIndex = i
				}
				depth++
			}
		}

		if ch == '}' && !inString && depth > 0 {
			depth--
			if depth == 0 {
				dictBraceIndex = -1
			}
		}
	}

	if depth > 0 {
		return dictBraceIndex
	}
	return -1
}

func (p *PythonPreprocessor) isDictionaryBrace(line string, braceIndex int) bool {
	before := strings.TrimSpace(line[:braceIndex])

	if strings.HasSuffix(before, ")") {
		return false
	}

	if strings.HasSuffix(before, "=") || strings.HasSuffix(before, ":") || strings.HasSuffix(before, "(") || strings.HasSuffix(before, "[") || strings.HasSuffix(before, ",") || strings.HasSuffix(before, "return") {
		return true
	}

	return false
}

func (p *PythonPreprocessor) ProcessReader(reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	first := true

	for scanner.Scan() {
		lines := p.processLine(scanner.Text())
		for _, line := range lines {
			if line != "" || !first {
				_, err := fmt.Fprintln(writer, line)
				if err != nil {
					return err
				}
			}
			first = false
		}
	}

	return scanner.Err()
}

func (p *PythonPreprocessor) ProcessFile(inputPath, outputPath string) error {
	p.indentLevel = 0
	p.structuralBlocks = 0
	p.dictDepth = 0

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("error opening input file: %v", err)
	}
	defer func() { _ = inputFile.Close() }()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer func() { _ = outputFile.Close() }()

	return p.ProcessReader(inputFile, outputFile)
}

func (p *PythonPreprocessor) ProcessString(input string) string {
	p.indentLevel = 0
	p.structuralBlocks = 0
	p.dictDepth = 0
	reader := strings.NewReader(input)
	var builder strings.Builder
	builder.Grow(len(input) + len(input)/4)
	_ = p.ProcessReader(reader, &builder)
	return builder.String()
}

func (p *PythonPreprocessor) IndentSize() int {
	return p.indentSize
}
