package bython

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go-Bython/internal/bython/diagnostic"
	"go-Bython/internal/bython/lexer"
	"go-Bython/internal/bython/token"
	"go-Bython/internal/bython/tokentest"
)

const (
	samplesDir = "../../samples"
)

var (
	fuzzSeeds = []string{
		"if x > 0 {\n    print(\"positive\");\n} else {\n    print(\"negative\");\n}",
		"def greet(name) {\n    return f\"Hello, {name}\";\n}",
		"config = {\n  \"name\": \"test\",\n  \"nested\": {\n    \"key\": \"value\"\n  }\n};",
		"squares = {x*x for x in range(10)};\nmapping = {k: v for k, v in items};",
		"def func(data={}, tags=set()) {\n    pass;\n}",
		"for hour in range(0,23) {\n    if (t == False) {\n        w('\\033[F')\n        } # up\n    s(1)\n    }",
		"def top() {\n    def second() {\n    m = {{1,2},{3,4}}\n    }\n}\n\ntop()",
		"try {\n    f();\n} except* E {\n    pass;\n} finally {\n}",
		"match cmd {\n    case \"go\" {\n        go();\n    }\n}",
		"@dec\n# c\nasync def f() { await g() }",
		"s = \"\"\"\n# not {\n\"\"\";\r\nx = f\"{x!r:>{w}}\";",
		"if a and \\\n    b {\n    pass;\n}",
		"x = (1 #c\r\"\"\"\n\"\"\")",
		"0%\"\"\r\"\" ",
		"0\\\n ;0",
	}
)

func FuzzLex(f *testing.F) {
	addSeeds(f)

	f.Fuzz(func(t *testing.T, src string) {
		s, err := lexer.Lex(src)
		if err != nil {
			checkSyntaxError(t, err)
			return
		}

		if err := tokentest.RoundTrip(s); err != nil {
			t.Fatalf("round trip failed for %q: %v", src, err)
		}
	})
}

func FuzzTranslate(f *testing.F) {
	addSeeds(f)

	f.Fuzz(func(t *testing.T, src string) {
		for _, size := range []int{2, 4} {
			out, err := Translate(src, size)
			if err != nil {
				checkSyntaxError(t, err)
				return
			}

			checkIndentation(t, src, out, size)
		}
	})
}

func addSeeds(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}

	err := filepath.WalkDir(samplesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f.Add(string(data))

		return nil
	})
	if err != nil {
		f.Fatal(err)
	}
}

func checkSyntaxError(t *testing.T, err error) {
	t.Helper()

	se, ok := errors.AsType[*diagnostic.SyntaxError](err)
	if !ok {
		t.Fatalf("error is not a *diagnostic.SyntaxError: %v", err)
	}

	if se.Line < 1 || se.Col < 1 || se.Msg == "" {
		t.Fatalf("malformed syntax error: %+v", se)
	}
}

func checkIndentation(t *testing.T, src, out string, size int) {
	t.Helper()

	s, err := lexer.Lex(out)
	if err != nil {
		t.Fatalf("output does not lex: %v\ninput: %q\noutput: %q", err, src, out)
	}

	atLineStart := true
	for _, tok := range s.Tokens {
		if tok.Kind == token.KindEOF {
			break
		}

		if atLineStart && tok.Kind != token.KindNewline {
			lead := s.Text[s.LineOf(tok).Start:tok.Start]
			for _, c := range []byte(lead) {
				if c != ' ' {
					t.Fatalf("line %d starts with %q\ninput: %q\noutput: %q", tok.Line, lead, src, out)
				}
			}

			if len(lead)%size != 0 {
				t.Fatalf("line %d indent %d is not a multiple of %d\ninput: %q\noutput: %q", tok.Line, len(lead), size, src, out)
			}
		}

		atLineStart = tok.Kind == token.KindNewline || tok.Kind == token.KindLineJoin
	}
}
