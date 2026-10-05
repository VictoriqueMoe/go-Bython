# go-Bython

A Go-based preprocessor that converts brace-style Python syntax to standard Python indentation.

## Overview

I hate Python, it's shit. and it's ugly. I found [Bython](https://github.com/mathialo/bython) and found it funny, but it had issues:
1. It was written in Python
2. It doesn't support all types of Python syntax, such as list comprehensions and generator expressions (or some shit)

So, I thought I would make it in go instead. 

go-Bython allows you to write Python code using braces `{}` instead of indentation, similar to languages like C, Java, or JavaScript. The preprocessor converts your brace-style code to standard Python with proper indentation.

Source is processed by a real lexer and parser, so a `{` is classified by its grammatical position (block, dict, set, or comprehension) and malformed input is reported with a precise `line:column` error instead of producing broken output.

## Features

- **Real lexer and parser** - Full Python tokenisation, including all string forms and nested f-strings (PEP 701)
- **Precise errors** - Unbalanced braces, unterminated strings, stray clauses, and similar mistakes fail with `file:line:column: message`, and no output file is written
- **Streaming** - Input is read, translated, and written one top-level statement at a time, so memory stays bounded by the largest top-level statement (one class or function), not by the file size
- **Fast concurrent processing** - Process multiple files in parallel using goroutines
- **Batch processing** - Convert entire directories recursively
- **Pattern matching** - Filter files by custom patterns (e.g., `*.py`, `*.pybrace`)
- **Configurable indentation** - Choose your preferred indent size
- **Source fidelity** - Expressions, comments, and string literals (including multi-line strings) are copied exactly; only block structure and trailing semicolons change

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
- Loops: `for`, `while`, including `for`/`else` and `while`/`else`
- Functions and classes: `def`, `class`, decorators, `async def`
- Exception handling: `try`, `except`, `except*`, `else`, `finally`
- Context managers: `with`, `async with`, parenthesised `with` items
- Pattern matching: `match` / `case` (and `match` still works as an ordinary name)
- Dictionaries, sets, and comprehensions, including multi-line and inside block headers (`if x == {} {`)
- One-line blocks: `if x { return 1 }`, nested `if a { if b { c() } }`
- Empty blocks: `def f() {}` becomes `def f():` with `pass`
- Brace on its own line (Allman style)
- Multiple statements per line separated by `;`
- Comments, including after braces (`} # note`)
- All string forms: raw, bytes, triple-quoted, f-strings and t-strings, with braces inside them left untouched
- `\` line continuations and multi-line bracketed expressions
- CRLF line endings, tabs, and a UTF-8 byte-order mark

## Caveats

### Brace-Style Only

This tool **only processes brace-style Python**. A colon-style block such as `if x:` is rejected with an error:

```text
input.py:1:5: expected '{' to open the 'if' block (Bython blocks use '{', not ':')
```

The same applies to files already written in standard Python and to files that mix the two styles. A colon followed by a brace (`if x: {`) is tolerated.

### Expressions Are Not Validated

Statements and block structure are fully parsed, but expressions are only checked for balanced brackets and obvious mistakes such as two values with no operator between them. Anything subtler (for example an invalid number like `1__2`) passes through and is reported by Python when the output runs.

## Streaming and Error Behaviour

Input is read in 64 KiB blocks and translated one top-level statement at a time, so memory depends on the largest statement, not the file size. A statement stays open while the next code line could still belong to it (`elif`, `else`, `except`, `finally`, an Allman `{`, or the target of a decorator). Comment and blank lines after a closing `}` are held until the next code line.

On error:

- `ProcessFile` writes nothing. Output goes to a temp file that is renamed into place only on success, so an existing output file is left untouched. Translating a file in place works.
- `ProcessReader` may already have written earlier statements, in 64 KiB blocks.
- `ProcessString` returns an empty string.

Errors are reported in reading order, so the first mistake in the file wins. Inputs of 2 GiB or more are rejected.

## Performance

Tested on AMD Ryzen 9 9950X3D (16-Core Processor).

### In-memory translation

| Benchmark                  | Time/op  | Memory/op | Allocs/op |
|----------------------------|----------|-----------|-----------|
| Simple if/else             | 720 ns   | 128 B     | 2         |
| Nested blocks (5 levels)   | 1.18 μs  | 256 B     | 2         |
| Class with methods         | 3.35 μs  | 737 B     | 2         |
| Complex program            | 7.06 μs  | 1.44 KB   | 2         |
| Large file (100 functions) | 83.6 μs  | 13.6 KB   | 2         |
| Process reader             | 795 ns   | 210 B     | 4         |
| String with braces         | 1.48 μs  | 288 B     | 2         |
| Parallel processing        | 177 ns   | 272 B     | 2         |

Memory/op counts new allocations per call, not peak memory. Lexer, parser, AST storage, and the output buffer are pooled and reused between calls, so steady-state translation makes only a couple of allocations regardless of file size.

### Whole files

Single-file mode on generated inputs (the `samples/` files concatenated), compared with the previous line-by-line processor. Peak heap is the largest heap size reported by `GODEBUG=gctrace=1`; the time is the tool's own reported processing time.

| Input | Previous: peak heap | Previous: time | Current: peak heap | Current: time |
|-------|---------------------|----------------|--------------------|---------------|
| 1 MB  | under 4 MB          | 122 ms         | under 4 MB         | 10 ms         |
| 10 MB | 3 MB                | 1200 ms        | 3 MB               | 135 ms        |
| 50 MB | 3 MB                | 6141 ms        | 3 MB               | 458 ms        |

Memory stays flat because only one top-level statement is held at a time (see [Streaming and Error Behaviour](#streaming-and-error-behaviour)). Most of the whole-file speed-up comes from buffered output: the previous processor issued one unbuffered write per output line, while the current one writes through a 64 KiB buffer.

Run benchmarks yourself:

```bash
go test -run '^$' -bench . -benchmem ./processor
```

## Testing

Run the tests:

```bash
go test ./internal/... ./processor/...
```

The lexer and translator also have fuzz tests. A plain `go test` runs them over their seed inputs and the `samples/` files; to fuzz for real after changing the lexer or parser:

```bash
go test ./internal/bython/ -run '^$' -fuzz FuzzTranslate -fuzztime 30s -parallel 4
```
