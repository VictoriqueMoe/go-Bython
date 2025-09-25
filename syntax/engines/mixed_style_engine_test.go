package engines

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMixedStyleEngine_DetectGoBythonFirst(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	//when - first line with go-Bython style
	err1 := engine.CheckSyntax("def foo() {", 1)
	//then
	assert.NoError(t, err1)

	//when - second line with standard Python style should error
	err2 := engine.CheckSyntax("def bar():", 2)
	//then
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "mixed syntax detected")
	assert.Contains(t, err2.Error(), "standard Python colon syntax found at line 2")
	assert.Contains(t, err2.Error(), "go-Bython braces were detected at line 1")
}

func TestMixedStyleEngine_DetectStandardPythonFirst(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	//when - first line with standard Python style
	err1 := engine.CheckSyntax("def foo():", 1)
	//then
	assert.NoError(t, err1)

	//when - second line with go-Bython style should error
	err2 := engine.CheckSyntax("def bar() {", 2)
	//then
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "mixed syntax detected")
	assert.Contains(t, err2.Error(), "go-Bython style brace found at line 2")
	assert.Contains(t, err2.Error(), "standard Python indentation was detected at line 1")
}

func TestMixedStyleEngine_ConsistentGoBythonStyle(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	//when - all go-Bython style
	err1 := engine.CheckSyntax("def foo() {", 1)
	err2 := engine.CheckSyntax("    if x > 0 {", 2)
	err3 := engine.CheckSyntax("        print('hello');", 3)
	err4 := engine.CheckSyntax("    }", 4)
	err5 := engine.CheckSyntax("}", 5)

	//then - no errors
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.NoError(t, err4)
	assert.NoError(t, err5)
}

func TestMixedStyleEngine_ConsistentStandardPythonStyle(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	//when - all standard Python style
	err1 := engine.CheckSyntax("def foo():", 1)
	err2 := engine.CheckSyntax("    if x > 0:", 2)
	err3 := engine.CheckSyntax("        print('hello')", 3)

	//then - no errors
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}

func TestMixedStyleEngine_SkipCommentsAndEmptyLines(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	//when - comments and empty lines should be ignored
	err1 := engine.CheckSyntax("# This is a comment", 1)
	err2 := engine.CheckSyntax("", 2)
	err3 := engine.CheckSyntax("   ", 3)
	err4 := engine.CheckSyntax("def foo() {", 4)
	err5 := engine.CheckSyntax("# Another comment", 5)

	//then - no errors
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.NoError(t, err4)
	assert.NoError(t, err5)
}

func TestMixedStyleEngine_DetectControlStatements(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	testCases := []struct {
		name string
		line string
	}{
		{"if statement", "if condition:"},
		{"elif statement", "elif other_condition:"},
		{"else statement", "else:"},
		{"while statement", "while True:"},
		{"for statement", "for i in range(10):"},
		{"def statement", "def function():"},
		{"class statement", "class MyClass:"},
		{"try statement", "try:"},
		{"except statement", "except Exception:"},
		{"finally statement", "finally:"},
		{"with statement", "with open('file') as f:"},
		{"main statement", "if __name__ == \"__main__\":"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			err := engine.CheckSyntax(tc.line, 1)
			//then
			assert.NoError(t, err)
		})
	}
}

func TestMixedStyleEngine_DetectStructuralBraces(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	testCases := []struct {
		name string
		line string
	}{
		{"function with brace", "def foo() {"},
		{"if with brace", "if condition {"},
		{"class with brace", "class MyClass {"},
		{"while with brace", "while True {"},
		{"for with brace", "for i in range(10) {"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			err := engine.CheckSyntax(tc.line, 1)
			//then
			assert.NoError(t, err)
		})
	}
}

func TestMixedStyleEngine_IgnoreDictionaryBraces(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	testCases := []struct {
		name string
		line string
	}{
		{"dictionary assignment", "data = {'key': 'value'}"},
		{"dictionary in function call", "func({'key': 'value'})"},
		{"dictionary return", "return {'key': 'value'}"},
		{"nested dictionary", "data = {'outer': {'inner': 'value'}}"},
		{"dictionary with colon", "config: {'setting': True}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			err := engine.CheckSyntax(tc.line, 1)
			//then
			assert.NoError(t, err, "Dictionary braces should not be detected as structural braces")
		})
	}
}

func TestMixedStyleEngine_IgnoreFStringBraces(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	testCases := []struct {
		name string
		line string
	}{
		{"f-string", "print(f'Hello {name}')"},
		{"F-string uppercase", "print(F'Value: {value}')"},
		{"nested f-string", "print(f'Outer {f\"Inner {x}\"}')"},
		{"complex f-string", "print(f'Result: {func(a, b)}')"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			err := engine.CheckSyntax(tc.line, 1)
			//then
			assert.NoError(t, err, "F-string braces should not be detected as structural braces")
		})
	}
}

func TestMixedStyleEngine_IgnoreStringLiterals(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	testCases := []struct {
		name string
		line string
	}{
		{"string with braces", "text = 'This has { and } braces'"},
		{"double quote string", "text = \"This has { and } braces\""},
		{"string with escaped quotes", "text = 'Don\\'t ignore { braces }'"},
		{"multiline string start", "text = '''Start {'''"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			err := engine.CheckSyntax(tc.line, 1)
			//then
			assert.NoError(t, err, "Braces inside strings should be ignored")
		})
	}
}

func TestMixedStyleEngine_Reset(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	//when - detect go-Bython style
	err1 := engine.CheckSyntax("def foo() {", 1)
	assert.NoError(t, err1)

	//when - reset engine
	engine.Reset()

	//when - should be able to detect standard Python style after reset
	err2 := engine.CheckSyntax("def bar():", 1)
	assert.NoError(t, err2)

	//when - should be able to detect go-Bython style after standard Python
	err3 := engine.CheckSyntax("def baz() {", 2)
	//then - should error because we detected standard Python first
	assert.Error(t, err3)
}

func TestMixedStyleEngine_LineNumbersInErrors(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()

	//when - establish standard Python at line 10
	err1 := engine.CheckSyntax("def foo():", 10)
	assert.NoError(t, err1)

	//when - try go-Bython at line 25
	err := engine.CheckSyntax("def bar() {", 25)

	//then - error should contain both line numbers
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "line 25")
	assert.Contains(t, err.Error(), "line 10")
}

func TestMixedStyleEngine_ComplexMixedScenario(t *testing.T) {
	//given
	engine := NewMixedStyleEngine()
	lines := []struct {
		line       string
		lineNumber int
		shouldErr  bool
	}{
		{"# Header comment", 1, false},
		{"", 2, false},
		{"def fibonacci(n):", 3, false},                  // Establish standard Python
		{"    if n <= 1:", 4, false},                     // Consistent standard Python
		{"        return n", 5, false},                   // Not a control statement
		{"    else:", 6, false},                          // Consistent standard Python
		{"        return fib(n-1) + fib(n-2)", 7, false}, // Not a control statement
		{"", 8, false},
		{"# This should cause an error", 9, false},
		{"def factorial(n) {", 10, true}, // Mixed! Should error
	}

	for _, test := range lines {
		//when
		err := engine.CheckSyntax(test.line, test.lineNumber)

		//then
		if test.shouldErr {
			assert.Error(t, err, "Expected error at line %d: %s", test.lineNumber, test.line)
			assert.Contains(t, err.Error(), "mixed syntax detected")
		} else {
			assert.NoError(t, err, "Unexpected error at line %d: %s", test.lineNumber, test.line)
		}
	}
}
