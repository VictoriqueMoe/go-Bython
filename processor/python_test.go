package processor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFibonacciAndCalculator(t *testing.T) {
	//given
	input := `# Go-style Python test file

def fibonacci(n) {
    if n <= 1 {
        return n;
    } else {
        return fibonacci(n-1) + fibonacci(n-2);
    }
}

class Calculator {
    def __init__(self, initial_value) {
        self.value = initial_value;
    }

    def add(self, x) {
        self.value += x;
        return self.value;
    }

    def multiply(self, x) {
        self.value *= x;
        return self.value;
    }
}

if __name__ == "__main__" {
    print("Starting program");

    # Test fibonacci
    for i in range(10) {
        print(f"fib({i}) = {fibonacci(i)}");
    }

    # Test calculator
    calc = Calculator(10);
    print(f"Initial: {calc.value}");
    print(f"After add 5: {calc.add(5)}");
    print(f"After multiply 2: {calc.multiply(2)}");

    # Nested conditions
    x = 15;
    if x > 10 {
        if x < 20 {
            print("x is between 10 and 20");
        } else {
            print("x is 20 or greater");
        }
    } else {
        print("x is 10 or less");
    }

    # While loop
    counter = 0;
    while counter < 5 {
        print(f"Counter: {counter}");
        counter += 1;
    }
}`

	expected := `# Go-style Python test file

def fibonacci(n):
  if n <= 1:
    return n
  else:
    return fibonacci(n-1) + fibonacci(n-2)

class Calculator:
  def __init__(self, initial_value):
    self.value = initial_value

  def add(self, x):
    self.value += x
    return self.value

  def multiply(self, x):
    self.value *= x
    return self.value

if __name__ == "__main__":
  print("Starting program")

  # Test fibonacci
  for i in range(10):
    print(f"fib({i}) = {fibonacci(i)}")

  # Test calculator
  calc = Calculator(10)
  print(f"Initial: {calc.value}")
  print(f"After add 5: {calc.add(5)}")
  print(f"After multiply 2: {calc.multiply(2)}")

  # Nested conditions
  x = 15
  if x > 10:
    if x < 20:
      print("x is between 10 and 20")
    else:
      print("x is 20 or greater")
  else:
    print("x is 10 or less")

  # While loop
  counter = 0
  while counter < 5:
    print(f"Counter: {counter}")
    counter += 1
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestSimpleIfElse(t *testing.T) {
	//given
	input := `if x > 0 {
    print("positive");
} else {
    print("negative");
}`

	expected := `if x > 0:
  print("positive")
else:
  print("negative")
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestNestedBlocks(t *testing.T) {
	//given
	input := `if a {
    if b {
        if c {
            print("deeply nested");
        }
    }
}`

	expected := `if a:
  if b:
    if c:
      print("deeply nested")
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestWhileLoop(t *testing.T) {
	//given
	input := `while True {
    print("loop");
    if break_condition {
        break;
    }
}`

	expected := `while True:
  print("loop")
  if break_condition:
    break
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestForLoop(t *testing.T) {
	//given
	input := `for i in range(10) {
    print(i);
}`

	expected := `for i in range(10):
  print(i)
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestFunctionDefinition(t *testing.T) {
	//given
	input := `def greet(name) {
    return f"Hello, {name}";
}`

	expected := `def greet(name):
  return f"Hello, {name}"
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestClassDefinition(t *testing.T) {
	//given
	input := `class Point {
    def __init__(self, x, y) {
        self.x = x;
        self.y = y;
    }
}`

	expected := `class Point:
  def __init__(self, x, y):
    self.x = x
    self.y = y
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestComments(t *testing.T) {
	//given
	input := `# This is a comment
if x {
    # Another comment
    print("hello");
}`

	expected := `# This is a comment
if x:
  # Another comment
  print("hello")
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestEmptyLines(t *testing.T) {
	//given
	input := `def foo() {
    print("test");

    print("after blank");
}`

	expected := `def foo():
  print("test")

  print("after blank")
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestDifferentIndentSize(t *testing.T) {
	//given
	input := `if x {
    print("test");
}`

	expected := `if x:
    print("test")
`

	p := NewPythonPreprocessor(4)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestTryExceptFinally(t *testing.T) {
	//given
	input := `try {
    risky_operation();
} except Exception {
    print("error");
} finally {
    cleanup();
}`

	expected := `try:
  risky_operation()
except Exception:
  print("error")
finally:
  cleanup()
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestWithStatement(t *testing.T) {
	//given
	input := `with open("file.txt") as f {
    data = f.read();
}`

	expected := `with open("file.txt") as f:
  data = f.read()
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestElifChain(t *testing.T) {
	//given
	input := `if x < 0 {
    print("negative");
} elif x == 0 {
    print("zero");
} elif x < 10 {
    print("small positive");
} else {
    print("large positive");
}`

	expected := `if x < 0:
  print("negative")
elif x == 0:
  print("zero")
elif x < 10:
  print("small positive")
else:
  print("large positive")
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestProcessReader(t *testing.T) {
	//given
	input := `if True {
    print("test");
}`

	expected := `if True:
  print("test")
`

	p := NewPythonPreprocessor(2)
	reader := strings.NewReader(input)
	var builder strings.Builder

	//when
	err := p.ProcessReader(reader, &builder)

	//then
	assert.NoError(t, err)
	assert.Equal(t, expected, builder.String())
}

func TestDictionaryCreation(t *testing.T) {
	//given
	input := `dict = {};
dict["key"] = "value";
dict["number"] = 42;`

	expected := `dict = {}
dict["key"] = "value"
dict["number"] = 42
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestDictionaryWithBraces(t *testing.T) {
	//given
	input := `config = {"name": "test", "value": 123};
data = {"users": ["alice", "bob"], "count": 2};`

	expected := `config = {"name": "test", "value": 123}
data = {"users": ["alice", "bob"], "count": 2}
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestDictionaryInControlFlow(t *testing.T) {
	//given
	input := `if user_data {
    settings = {};
    settings["theme"] = "dark";
    result = {"status": "ok", "data": settings};
}`

	expected := `if user_data:
  settings = {}
  settings["theme"] = "dark"
  result = {"status": "ok", "data": settings}
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestDictionaryWithFStrings(t *testing.T) {
	//given
	input := `user = {"name": "Alice", "age": 30};
for key in user {
    print(f"Key: {key}, Value: {user[key]}");
}`

	expected := `user = {"name": "Alice", "age": 30}
for key in user:
  print(f"Key: {key}, Value: {user[key]}")
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestMultilineDictionary(t *testing.T) {
	//given
	input := `config = {
  "name": "test",
  "value": 123,
  "nested": {
    "key": "value"
  }
};`

	expected := `config = {
  "name": "test",
  "value": 123,
  "nested": {
    "key": "value"
  }
}
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestSetLiterals(t *testing.T) {
	//given
	input := `my_set = {1, 2, 3, 4};
numbers = {10, 20, 30};
empty = set();`

	expected := `my_set = {1, 2, 3, 4}
numbers = {10, 20, 30}
empty = set()
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestMultilineSet(t *testing.T) {
	//given
	input := `tags = {
  "python",
  "golang",
  "rust"
};`

	expected := `tags = {
  "python",
  "golang",
  "rust"
}
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestSetComprehension(t *testing.T) {
	//given
	input := `squares = {x*x for x in range(10)};
evens = {n for n in numbers if n % 2 == 0};`

	expected := `squares = {x*x for x in range(10)}
evens = {n for n in numbers if n % 2 == 0}
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestDictComprehension(t *testing.T) {
	//given
	input := `mapping = {k: v for k, v in items};
squares_dict = {x: x*x for x in range(5)};`

	expected := `mapping = {k: v for k, v in items}
squares_dict = {x: x*x for x in range(5)}
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestDictSetAsFunctionArgs(t *testing.T) {
	//given
	input := `result = process_data(items={1, 2, 3}, config={"key": "value"});
output = func(tags={"a", "b"}, data={"x": 1});`

	expected := `result = process_data(items={1, 2, 3}, config={"key": "value"})
output = func(tags={"a", "b"}, data={"x": 1})
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestDefaultParameters(t *testing.T) {
	//given
	input := `def func(data={}, tags=set()) {
    pass;
}

def process(config={"default": True}, items={1, 2}) {
    return config;
}`

	expected := `def func(data={}, tags=set()):
  pass

def process(config={"default": True}, items={1, 2}):
  return config
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestNestedDictSet(t *testing.T) {
	//given
	input := `data = {
  "users": {1, 2, 3},
  "config": {
    "nested": {"key": "value"}
  }
};`

	expected := `data = {
  "users": {1, 2, 3},
  "config": {
    "nested": {"key": "value"}
  }
}
`

	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)
	assert.NoError(t, err)

	//then
	assert.Equal(t, expected, result)
}

func TestMixedSyntaxDetection_StandardPythonFirst(t *testing.T) {
	//given
	input := `# Standard Python first
def one():
    for i in range(5):
        print(i)

# go-Bython second
def two() {
    for i in range(5) {
        print(i);
    }
}`

	p := NewPythonPreprocessor(2)
	//when
	_, err := p.ProcessString(input)
	//then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mixed syntax detected")
	assert.Contains(t, err.Error(), "go-Bython style brace found at line 7")
	assert.Contains(t, err.Error(), "standard Python indentation was detected at line 2")
}

func TestMixedSyntaxDetection_GoBythonFirst(t *testing.T) {
	//given
	input := `# go-Bython first
def one() {
    for i in range(5) {
        print(i);
    }
}

# Standard Python second
def two():
    for i in range(5):
        print(i)`

	p := NewPythonPreprocessor(2)
	//when
	_, err := p.ProcessString(input)
	//then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mixed syntax detected")
	assert.Contains(t, err.Error(), "standard Python colon syntax found at line 9")
	assert.Contains(t, err.Error(), "go-Bython braces were detected at line 2")
}

func TestPureGoBythonSyntax_NoError(t *testing.T) {
	//given
	input := `# Pure go-Bython - should work fine
def fibonacci(n) {
    if n <= 1 {
        return n;
    } else {
        return fibonacci(n-1) + fibonacci(n-2);
    }
}

class Calculator {
    def __init__(self, initial_value) {
        self.value = initial_value;
    }

    def add(self, x) {
        self.value += x;
        return self.value;
    }
}`

	expected := `# Pure go-Bython - should work fine
def fibonacci(n):
  if n <= 1:
    return n
  else:
    return fibonacci(n-1) + fibonacci(n-2)

class Calculator:
  def __init__(self, initial_value):
    self.value = initial_value

  def add(self, x):
    self.value += x
    return self.value
`

	p := NewPythonPreprocessor(2)
	//when
	result, err := p.ProcessString(input)
	//then
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestPureStandardPython_NoError(t *testing.T) {
	//given - Pure standard Python (no braces, so no transformation expected)
	input := `# Pure standard Python - should work fine
def fibonacci(n):
    if n <= 1:
        return n
    else:
        return fibonacci(n-1) + fibonacci(n-2)

class Calculator:
    def __init__(self, initial_value):
        self.value = initial_value

    def add(self, x):
        self.value += x
        return self.value`

	expected := `# Pure standard Python - should work fine
def fibonacci(n):
if n <= 1:
return n
else:
return fibonacci(n-1) + fibonacci(n-2)

class Calculator:
def __init__(self, initial_value):
self.value = initial_value

def add(self, x):
self.value += x
return self.value
`

	p := NewPythonPreprocessor(2)
	//when
	result, err := p.ProcessString(input)
	//then
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

// Detailed mixed syntax detection tests
func TestMixedSyntax_DetectGoBythonFirst(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

	//when - first line with go-Bython style
	err1 := p.checkMixedSyntax("def foo() {", 1)
	//then
	assert.NoError(t, err1)

	//when - second line with standard Python style should error
	err2 := p.checkMixedSyntax("def bar():", 2)
	//then
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "mixed syntax detected")
	assert.Contains(t, err2.Error(), "standard Python colon syntax found at line 2")
	assert.Contains(t, err2.Error(), "go-Bython braces were detected at line 1")
}

func TestMixedSyntax_DetectStandardPythonFirst(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

	//when - first line with standard Python style
	err1 := p.checkMixedSyntax("def foo():", 1)
	//then
	assert.NoError(t, err1)

	//when - second line with go-Bython style should error
	err2 := p.checkMixedSyntax("def bar() {", 2)
	//then
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "mixed syntax detected")
	assert.Contains(t, err2.Error(), "go-Bython style brace found at line 2")
	assert.Contains(t, err2.Error(), "standard Python indentation was detected at line 1")
}

func TestMixedSyntax_ConsistentGoBythonStyle(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

	//when - all go-Bython style
	err1 := p.checkMixedSyntax("def foo() {", 1)
	err2 := p.checkMixedSyntax("    if x > 0 {", 2)
	err3 := p.checkMixedSyntax("        print('hello');", 3)
	err4 := p.checkMixedSyntax("    }", 4)
	err5 := p.checkMixedSyntax("}", 5)

	//then - no errors
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.NoError(t, err4)
	assert.NoError(t, err5)
}

func TestMixedSyntax_ConsistentStandardPythonStyle(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

	//when - all standard Python style
	err1 := p.checkMixedSyntax("def foo():", 1)
	err2 := p.checkMixedSyntax("    if x > 0:", 2)
	err3 := p.checkMixedSyntax("        print('hello')", 3)

	//then - no errors
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}

func TestMixedSyntax_SkipCommentsAndEmptyLines(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

	//when - comments and empty lines should be ignored
	err1 := p.checkMixedSyntax("# This is a comment", 1)
	err2 := p.checkMixedSyntax("", 2)
	err3 := p.checkMixedSyntax("   ", 3)
	err4 := p.checkMixedSyntax("def foo() {", 4)
	err5 := p.checkMixedSyntax("# Another comment", 5)

	//then - no errors
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.NoError(t, err4)
	assert.NoError(t, err5)
}

func TestMixedSyntax_DetectControlStatements(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

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
			err := p.checkMixedSyntax(tc.line, 1)
			//then
			assert.NoError(t, err)
		})
	}
}

func TestMixedSyntax_DetectStructuralBraces(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

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
			err := p.checkMixedSyntax(tc.line, 1)
			//then
			assert.NoError(t, err)
		})
	}
}

func TestMixedSyntax_IgnoreDictionaryBraces(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

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
			err := p.checkMixedSyntax(tc.line, 1)
			//then
			assert.NoError(t, err, "Dictionary braces should not be detected as structural braces")
		})
	}
}

func TestMixedSyntax_IgnoreFStringBraces(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

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
			err := p.checkMixedSyntax(tc.line, 1)
			//then
			assert.NoError(t, err, "F-string braces should not be detected as structural braces")
		})
	}
}

func TestMixedSyntax_IgnoreStringLiterals(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

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
			err := p.checkMixedSyntax(tc.line, 1)
			//then
			assert.NoError(t, err, "Braces inside strings should be ignored")
		})
	}
}

func TestMixedSyntax_ProcessStringReset(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4)

	//when - process a go-Bython style string
	result1, err1 := p.ProcessString("def foo() {\n    print('hello')\n}")
	assert.NoError(t, err1)
	assert.NotEmpty(t, result1)

	//when - process a standard Python style string (should work after reset)
	result2, err2 := p.ProcessString("def bar():\n    print('world')")
	assert.NoError(t, err2)
	assert.NotEmpty(t, result2)
}

func TestMixedSyntax_LineNumbersInErrors(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)

	//when - establish standard Python at line 10
	err1 := p.checkMixedSyntax("def foo():", 10)
	assert.NoError(t, err1)

	//when - try go-Bython at line 25
	err := p.checkMixedSyntax("def bar() {", 25)

	//then - error should contain both line numbers
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "line 25")
	assert.Contains(t, err.Error(), "line 10")
}

func TestMixedSyntax_ComplexMixedScenario(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4).(*PythonPreprocessor)
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
		err := p.checkMixedSyntax(test.line, test.lineNumber)

		//then
		if test.shouldErr {
			assert.Error(t, err, "Expected error at line %d: %s", test.lineNumber, test.line)
			assert.Contains(t, err.Error(), "mixed syntax detected")
		} else {
			assert.NoError(t, err, "Unexpected error at line %d: %s", test.lineNumber, test.line)
		}
	}
}

func TestMixedSyntax_FullProcessingWithMixedSyntax(t *testing.T) {
	//given
	p := NewPythonPreprocessor(4)
	input := `def foo():
    print('standard python')

def bar() {
    print('go-bython style')
}`

	//when
	_, err := p.ProcessString(input)

	//then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mixed syntax detected")
}
