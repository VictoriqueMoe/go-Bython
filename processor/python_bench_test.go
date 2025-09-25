package processor

import (
	"strings"
	"testing"
)

func BenchmarkSimpleIfElse(b *testing.B) {
	input := `if x > 0 {
    print("positive");
} else {
    print("negative");
}`

	p := NewPythonPreprocessor(2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := p.ProcessString(input)
		_ = result
	}
}

func BenchmarkNestedBlocks(b *testing.B) {
	input := `if a {
    if b {
        if c {
            if d {
                if e {
                    print("deeply nested");
                }
            }
        }
    }
}`

	p := NewPythonPreprocessor(2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := p.ProcessString(input)
		_ = result
	}
}

func BenchmarkClassWithMethods(b *testing.B) {
	input := `class Calculator {
    def __init__(self, initial_value) {
        self.value = initial_value;
    }

    def add(self, x) {
        self.value += x;
        return self.value;
    }

    def subtract(self, x) {
        self.value -= x;
        return self.value;
    }

    def multiply(self, x) {
        self.value *= x;
        return self.value;
    }

    def divide(self, x) {
        if x == 0 {
            raise ValueError("Cannot divide by zero");
        }
        self.value /= x;
        return self.value;
    }
}`

	p := NewPythonPreprocessor(2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := p.ProcessString(input)
		_ = result
	}
}

func BenchmarkComplexProgram(b *testing.B) {
	input := `def fibonacci(n) {
    if n <= 0 {
        return 0;
    } elif n == 1 {
        return 1;
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

    for i in range(10) {
        print(f"fib({i}) = {fibonacci(i)}");
    }

    calc = Calculator(10);
    print(f"Initial: {calc.value}");
    print(f"After add 5: {calc.add(5)}");
    print(f"After multiply 2: {calc.multiply(2)}");

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

    counter = 0;
    while counter < 5 {
        print(f"Counter: {counter}");
        counter += 1;
    }
}`

	p := NewPythonPreprocessor(2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := p.ProcessString(input)
		_ = result
	}
}

func BenchmarkLargeFile(b *testing.B) {
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteString(`def function`)
		builder.WriteString(string(rune('a' + (i % 26))))
		builder.WriteString(`(x) {
    if x > 0 {
        for i in range(10) {
            print(i);
        }
    }
}

`)
	}

	input := builder.String()
	p := NewPythonPreprocessor(2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := p.ProcessString(input)
		_ = result
	}
}

func BenchmarkProcessReader(b *testing.B) {
	input := `if x > 0 {
    print("positive");
} else {
    print("negative");
}`

	p := NewPythonPreprocessor(2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(input)
		var builder strings.Builder
		_ = p.ProcessReader(reader, &builder)
	}
}

func BenchmarkStringWithBraces(b *testing.B) {
	input := `def greet(name) {
    return f"Hello, {name}! Welcome to {place}!";
}

class Person {
    def __init__(self, name) {
        self.name = name;
        self.data = {"key": "value", "count": 42};
    }
}`

	p := NewPythonPreprocessor(2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := p.ProcessString(input)
		_ = result
	}
}

func BenchmarkParallel(b *testing.B) {
	input := `class Calculator {
    def __init__(self, initial_value) {
        self.value = initial_value;
    }

    def add(self, x) {
        self.value += x;
        return self.value;
    }
}`

	b.RunParallel(func(pb *testing.PB) {
		p := NewPythonPreprocessor(2)
		for pb.Next() {
			result, _ := p.ProcessString(input)
			_ = result
		}
	})
}
