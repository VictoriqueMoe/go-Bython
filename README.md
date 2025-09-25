# go-Bython

A Go-based preprocessor that converts brace-style Python syntax to standard Python indentation.

## Overview

Inspired by [Bython](https://github.com/mathialo/bython),
go-Bython allows you to write Python code using braces `{}` instead of indentation, similar to languages like C, Java,
or JavaScript. The preprocessor automatically converts your brace-style code to standard Python with proper indentation.

## Features

- **Fast concurrent processing** - Process multiple files in parallel using goroutines
- **Batch processing** - Convert entire directories recursively
- **Pattern matching** - Filter files by custom patterns (e.g., `*.py`, `*.pybrace`)
- **Configurable indentation** - Choose your preferred indent size (2 or 4 spaces)
- **Smart brace detection** - Ignores braces inside strings and f-strings
- **Supports all Python constructs** - if/elif/else, loops, functions, classes, try/except, with statements

## Installation

```bash
go install
```

Or build from source:

```bash
go build -o go-bython
```

## Usage

### Single File Mode

Convert a single file:

```bash
go-bython -i input.py -o output.py
```

With custom indentation:

```bash
go-bython -i input.py -o output.py -indent 4
```

### Batch Processing Mode

Convert an entire directory:

```bash
go-bython -d ./src -od ./output
```

With custom file pattern and worker count:

```bash
go-bython -d ./src -od ./output -pattern "*.pybrace" -workers 8
```

### Command Line Options

- `-i` - Input file path (required for single file mode)
- `-o` - Output file path (required for single file mode)
- `-d` - Input directory for batch processing
- `-od` - Output directory for batch processing
- `-pattern` - File pattern to match (default: `*.py`)
- `-workers` - Number of concurrent workers (default: 4)
- `-indent` - Number of spaces for indentation (default: 2)

## Quick Start

Try it out with the included sample files:

```bash
# Process the sample files
go run main.go -d ./samples -od ./output

# Or build and run
go build -o go-bython
./go-bython -d ./samples -od ./output
```

The sample files demonstrate various Python constructs converted from brace-style to standard indentation.

## Example

### Input (Brace-style Python):

```python
class Calculator {
    def __init__(self, initial_value) {
        self.value = initial_value;
    }

    def add(self, x) {
        self.value += x;
        return self.value;
    }
}

if __name__ == "__main__" {
    calc = Calculator(10);

    for i in range(5) {
        print(f"Adding {i}: {calc.add(i)}");
    }
}
```

### Output (Standard Python):

```python
class Calculator:
  def __init__(self, initial_value):
    self.value = initial_value

  def add(self, x):
    self.value += x
    return self.value

if __name__ == "__main__":
  calc = Calculator(10)

  for i in range(5):
    print(f"Adding {i}: {calc.add(i)}")
```

## Supported Python Constructs

- Control flow: `if`, `elif`, `else`
- Loops: `for`, `while`
- Functions: `def`
- Classes: `class`
- Exception handling: `try`, `except`, `finally`
- Context managers: `with`
- Dictionaries and sets (including multiline)
- Dict/set comprehensions
- Comments
- F-strings and string literals
- Nested blocks

## Caveats

### Opening Brace Style

The opening brace `{` **must be on the same line** as the control statement, not on a new line.

**Supported (K&R/1TBS style):**
```python
if condition {
    statement;
}
```

**Not supported (Allman style):**
```python
if condition
{
    statement;
}
```

### Single-Line Statements

Single-line control statements with braces on the same line are **not supported**.

**Not supported:**
```python
if (firsttick == False) {sys.stdout.write('\033[F')}
```

**Workaround - use multiple lines:**
```python
if (firsttick == False) {
    sys.stdout.write('\033[F');
}
```

### Brace-Style Only

This tool **only processes brace-style Python** and converts it to standard Python indentation. It does not process files that are already in standard Python format.

**Input must use brace-style syntax:**
```python
def function_one() {
    print("Using braces");
}

def function_two() {
    print("Also using braces");
    if condition {
        return True;
    }
}
```

**Not supported - standard Python input:**
```python
# This won't be processed correctly - it's already standard Python!
def function_one():
    print("Standard Python")

def function_two():
    print("Also standard Python")
    if condition:
        return True
```

**Not supported - mixing styles in one file:**
```python
# Brace style
def function_one() {
    print("Using braces");
}

# Standard Python style mixed in - will cause issues!
def function_two():
    print("Using colons and indentation")
    if condition:
        return True
```

## Architecture

```
go-Bython/
├── main.go                 # CLI entry point
├── processor/
│   ├── processor.go        # Processor interface
│   ├── python.go          # Python preprocessor implementation
│   ├── python_test.go     # Unit tests
│   ├── folder.go          # Folder/batch processing
│   └── folder_test.go     # Folder processing tests
└── README.md
```

## Performance

go-Bython is designed for speed and efficiency, using concurrency to process files in parallel.

### Benchmark Results

Tested on AMD Ryzen 9 9950X3D (16-Core Processor):

| Benchmark                  | Time/op | Memory/op | Allocs/op |
|----------------------------|---------|-----------|-----------|
| Simple if/else             | 989 ns  | 4.46 KB   | 16        |
| Nested blocks (5 levels)   | 1.22 μs | 4.74 KB   | 27        |
| Class with methods         | 2.68 μs | 6.26 KB   | 68        |
| Complex program            | 5.22 μs | 8.60 KB   | 143       |
| Large file (100 functions) | 49.0 μs | 44.2 KB   | 1504      |
| String with braces         | 1.49 μs | 5.04 KB   | 29        |
| Parallel processing        | 613 ns  | 4.95 KB   | 27        |

**So... this means:**

- **~1 million simple statements/second** on a single core
- **~192,000 complex statements/second** with nested structures
- **~20,400 functions/second** for large files
- **Efficient f-string handling** with braces in strings
- **Linear scalability** with concurrent processing

### Real-world Performance

- A typical 100-line Python file processes in **~10 microseconds**
- A 1000-file codebase can be processed in **under 1 second** with 8 workers
- Memory efficient: ~44KB per 100 functions
- Optimised string processing with pre-allocated buffers

Run benchmarks yourself:

```bash
go test -bench=. -benchmem ./processor
```

## Testing

Run all tests:

```bash
go test ./...
```

Run with verbose output:

```bash
go test ./... -v
```