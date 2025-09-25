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
