package engines

import (
	"fmt"
	"strings"
)

// SyntaxStyle represents the detected syntax style
type SyntaxStyle int

const (
	StyleUnknown SyntaxStyle = iota
	StyleStandardPython
	StyleGoBython
)

// MixedStyleEngine detects when both standard Python and go-Bython styles are mixed
type MixedStyleEngine struct {
	detectedStyle      SyntaxStyle
	standardPythonLine int
	goBythonLine       int
	// We need access to some helper functions
	helper *syntaxHelper
}

// syntaxHelper provides helper functions for syntax detection
type syntaxHelper struct {
	controlKeywords []string
}

// NewMixedStyleEngine creates a new mixed style detection engine
func NewMixedStyleEngine() *MixedStyleEngine {
	helper := &syntaxHelper{
		controlKeywords: []string{
			"if ", "elif ", "else", "while ", "for ", "def ", "class ", "try", "except", "finally", "with ",
		},
	}

	return &MixedStyleEngine{
		detectedStyle:      StyleUnknown,
		standardPythonLine: 0,
		goBythonLine:       0,
		helper:             helper,
	}
}

// CheckSyntax implements syntax.Engine interface
func (e *MixedStyleEngine) CheckSyntax(line string, lineNumber int) error {
	trimmed := strings.TrimSpace(line)

	// Skip empty lines and comments
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}

	// Check for go-Bython style (braces)
	hasStructuralBrace := e.helper.findStructuralBrace(trimmed) != -1
	if hasStructuralBrace {
		if e.detectedStyle == StyleStandardPython {
			return fmt.Errorf("mixed syntax detected: go-Bython style brace found at line %d, but standard Python indentation was detected at line %d", lineNumber, e.standardPythonLine)
		}
		if e.detectedStyle == StyleUnknown {
			e.detectedStyle = StyleGoBython
			e.goBythonLine = lineNumber
		}
		return nil
	}

	// Check for standard Python style (control statements with colons and indented blocks)
	if e.helper.isControlStatement(trimmed) && strings.HasSuffix(trimmed, ":") {
		if e.detectedStyle == StyleGoBython {
			return fmt.Errorf("mixed syntax detected: standard Python colon syntax found at line %d, but go-Bython braces were detected at line %d", lineNumber, e.goBythonLine)
		}
		if e.detectedStyle == StyleUnknown {
			e.detectedStyle = StyleStandardPython
			e.standardPythonLine = lineNumber
		}
		return nil
	}

	return nil
}

// Reset clears the internal state
func (e *MixedStyleEngine) Reset() {
	e.detectedStyle = StyleUnknown
	e.standardPythonLine = 0
	e.goBythonLine = 0
}

// Helper methods

func (h *syntaxHelper) isControlStatement(line string) bool {
	for _, keyword := range h.controlKeywords {
		if strings.HasPrefix(line, keyword) || line == strings.TrimSpace(keyword) {
			return true
		}
	}

	if strings.Contains(line, "__main__") {
		return true
	}

	return false
}

func (h *syntaxHelper) findStructuralBrace(line string) int {
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
			if h.isDictionaryBrace(line, i) {
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

func (h *syntaxHelper) isDictionaryBrace(line string, braceIndex int) bool {
	before := strings.TrimSpace(line[:braceIndex])

	if strings.HasSuffix(before, ")") {
		return false
	}

	if strings.HasSuffix(before, "=") || strings.HasSuffix(before, ":") || strings.HasSuffix(before, "(") || strings.HasSuffix(before, "[") || strings.HasSuffix(before, ",") || strings.HasSuffix(before, "return") {
		return true
	}

	return false
}
