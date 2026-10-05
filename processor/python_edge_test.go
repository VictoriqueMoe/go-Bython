package processor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var edgeCases = []struct {
	name     string
	input    string
	expected string
}{
	{"empty_dict_compare_in_if_header", "if x == {} {\n    y = 1;\n}", "if x == {}:\n  y = 1\n"},
	{"dict_literal_after_in_in_for_header", "for k in {1: 2} {\n    print(k);\n}", "for k in {1: 2}:\n  print(k)\n"},
	{"set_literal_at_start_of_if_condition", "if {1,2} & s {\n    print(1);\n}", "if {1,2} & s:\n  print(1)\n"},
	{"dict_after_not_in_if_header", "if not {} {\n    pass;\n}", "if not {}:\n  pass\n"},
	{"dict_after_in_operator_in_if_header", "if k in {\"a\": 1} {\n    pass;\n}", "if k in {\"a\": 1}:\n  pass\n"},
	{"set_comprehension_in_for_header", "for x in {y for y in z} {\n    pass;\n}", "for x in {y for y in z}:\n  pass\n"},
	{"set_literal_in_elif_header", "if a {\n    pass;\n} elif {1} <= s {\n    pass;\n}", "if a:\n  pass\nelif {1} <= s:\n  pass\n"},
	{"dict_inside_call_in_while_header", "while d.get('a', {}) {\n    break;\n}", "while d.get('a', {}):\n  break\n"},
	{"dict_in_subscript_in_header", "if d[{1}] {\n    pass;\n}", "if d[{1}]:\n  pass\n"},
	{"lambda_returning_set_in_for_header", "for k in sorted(d, key=lambda k: {k}) {\n    pass;\n}", "for k in sorted(d, key=lambda k: {k}):\n  pass\n"},
	{"lambda_dict_in_if_header", "if (lambda: {})() {\n    pass;\n}", "if (lambda: {})():\n  pass\n"},
	{"lambda_display_before_block", "if lambda: {} {\n    pass;\n}", "if lambda: {}:\n  pass\n"},
	{"multiline_dict_in_for_header", "for k in {\n    1, 2\n} {\n    pass;\n}", "for k in {\n    1, 2\n}:\n  pass\n"},
	{"multiline_call_with_dict_in_if_header", "if check({\n    \"a\": 1\n}) {\n    pass;\n}", "if check({\n    \"a\": 1\n}):\n  pass\n"},
	{"dict_literal_expression_statement_top_level", "{\n    \"a\": 1\n}.items();\nx = 1;", "{\n    \"a\": 1\n}.items()\nx = 1\n"},
	{"dict_literal_expression_statement_in_block", "if x {\n    {\n        \"a\": 1\n    }.items();\n    y();\n}\nz();", "if x:\n  {\n      \"a\": 1\n  }.items()\n  y()\nz()\n"},
	{"single_line_dict_expression_statement", "{\"a\": 1}.get(\"a\");\nx = 1;", "{\"a\": 1}.get(\"a\")\nx = 1\n"},
	{"multiline_dict_after_else_in_conditional_expr", "def f() {\n    x = a if c else {\n        1: 2\n    };\n    return x;\n}", "def f():\n  x = a if c else {\n      1: 2\n  }\n  return x\n"},
	{"multiline_dict_after_yield", "def g() {\n    yield {\n        1: 2\n    };\n}", "def g():\n  yield {\n      1: 2\n  }\n"},
	{"multiline_set_after_binary_operator_in_block", "def f() {\n    s = a | {\n        1, 2\n    };\n    return s;\n}", "def f():\n  s = a | {\n      1, 2\n  }\n  return s\n"},
	{"identifier_with_keyword_prefix_try", "tryset = base | {\n    1, 2\n};\nx = 1;", "tryset = base | {\n    1, 2\n}\nx = 1\n"},
	{"main_string_hack_false_positive", "x = d.get(\"__main__\") or {\n  \"a\": 1\n};", "x = d.get(\"__main__\") or {\n  \"a\": 1\n}\n"},
	{"if_main_guard", "if __name__ == \"__main__\" {\n    main();\n}", "if __name__ == \"__main__\":\n  main()\n"},
	{"lambda_with_colon_statement", "f = lambda x: x + 1;\nif f(1) {\n    pass;\n}", "f = lambda x: x + 1\nif f(1):\n  pass\n"},
	{"lambda_returning_dict", "f = lambda: {};\ng = lambda: {1: 2};", "f = lambda: {}\ng = lambda: {1: 2}\n"},
	{"decorator_with_dict_argument", "@register({\"a\": 1})\ndef f() {\n    return 1;\n}", "@register({\"a\": 1})\ndef f():\n  return 1\n"},
	{"multiline_decorator", "@app.route(\n    \"/\"\n)\ndef f() {\n    pass;\n}", "@app.route(\n    \"/\"\n)\ndef f():\n  pass\n"},
	{"decorator_comment_gap", "@dec\n# c\ndef f() {\n    pass;\n}", "@dec\n# c\ndef f():\n  pass\n"},
	{"async_def", "async def f() {\n    await g();\n}", "async def f():\n  await g()\n"},
	{"async_for_and_async_with", "async def f() {\n    async for x in y {\n        pass;\n    }\n    async with a as b {\n        pass;\n    }\n}", "async def f():\n  async for x in y:\n    pass\n  async with a as b:\n    pass\n"},
	{"async_one_line_block", "async def f() { await g() }", "async def f():\n  await g()\n"},
	{"match_case_soft_keywords", "match cmd {\n    case \"go\" {\n        go();\n    }\n    case _ {\n        pass;\n    }\n}", "match cmd:\n  case \"go\":\n    go()\n  case _:\n    pass\n"},
	{"match_case_with_mapping_pattern", "match d {\n    case {\"k\": v} {\n        print(v);\n    }\n}", "match d:\n  case {\"k\": v}:\n    print(v)\n"},
	{"match_used_as_identifier", "match = re.match(p, s);\nif match {\n    print(match);\n}", "match = re.match(p, s)\nif match:\n  print(match)\n"},
	{"match_identifier_method_call_with_brace_block", "if match.group(1) {\n    pass;\n}", "if match.group(1):\n  pass\n"},
	{"match_call_statement", "match(x);", "match(x)\n"},
	{"except_star", "try {\n    f();\n} except* ValueError {\n    pass;\n}", "try:\n  f()\nexcept* ValueError:\n  pass\n"},
	{"except_tuple_as", "try {\n    f();\n} except (A, B) as e {\n    print(e);\n}", "try:\n  f()\nexcept (A, B) as e:\n  print(e)\n"},
	{"bare_except", "try {\n    f();\n} except {\n    pass;\n}", "try:\n  f()\nexcept:\n  pass\n"},
	{"for_else", "for x in y {\n    pass;\n} else {\n    done();\n}", "for x in y:\n  pass\nelse:\n  done()\n"},
	{"while_else", "while x {\n    x -= 1;\n} else {\n    done();\n}", "while x:\n  x -= 1\nelse:\n  done()\n"},
	{"try_except_else_finally", "try {\n    f();\n} except E {\n    g();\n} else {\n    h();\n} finally {\n    i();\n}", "try:\n  f()\nexcept E:\n  g()\nelse:\n  h()\nfinally:\n  i()\n"},
	{"else_without_spaces", "if x {\n    a();\n}else{\n    b();\n}", "if x:\n  a()\nelse:\n  b()\n"},
	{"keyword_followed_by_paren_without_space", "while(x) {\n    x -= 1;\n}", "while(x):\n  x -= 1\n"},
	{"if_paren_no_space_brace_no_space", "if(x){\n    pass;\n}", "if(x):\n  pass\n"},
	{"class_with_bases_and_metaclass", "class A(B, metaclass=M) {\n    pass;\n}", "class A(B, metaclass=M):\n  pass\n"},
	{"class_keyword_arg_with_dict_default", "class A(B, opts={}) {\n    pass;\n}", "class A(B, opts={}):\n  pass\n"},
	{"return_annotation_dict", "def f() -> dict {\n    return {};\n}", "def f() -> dict:\n  return {}\n"},
	{"param_annotation_and_dict_default", "def f(x: dict = {}) -> None {\n    pass;\n}", "def f(x: dict = {}) -> None:\n  pass\n"},
	{"pep695_generic_def", "def f[T](x: T) -> T {\n    return x;\n}", "def f[T](x: T) -> T:\n  return x\n"},
	{"pep695_type_statement", "type Point = tuple[float, float];\nif x {\n    pass;\n}", "type Point = tuple[float, float]\nif x:\n  pass\n"},
	{"walrus_in_header", "if (n := len(a)) > 10 {\n    print(n);\n}", "if (n := len(a)) > 10:\n  print(n)\n"},
	{"one_line_block", "def f(x) {\n    if x { return 1 }\n    return 2;\n}", "def f(x):\n  if x:\n    return 1\n  return 2\n"},
	{"one_line_block_multiple_statements", "if x { a(); b() }", "if x:\n  a(); b()\n"},
	{"nested_one_line_blocks", "if x { if y { z() } }", "if x:\n  if y:\n    z()\n"},
	{"one_line_block_returning_dict", "def f() { return {1: 2} }\nx = 1;", "def f():\n  return {1: 2}\nx = 1\n"},
	{"one_line_block_ellipsis", "def f() { ... }", "def f():\n  ...\n"},
	{"empty_block_same_line", "def f() {}\nx = 1;", "def f():\n  pass\nx = 1\n"},
	{"empty_block_split_lines", "def f() {\n}\nx = 1;", "def f():\n  pass\nx = 1\n"},
	{"empty_class_body", "class E(Exception) {}", "class E(Exception):\n  pass\n"},
	{"block_with_only_semicolon", "if x {\n    ;\n}", "if x:\n  pass\n"},
	{"comment_only_block", "if x {\n    # todo\n}", "if x:\n  # todo\n  pass\n"},
	{"multiple_statements_per_line", "a = 1; b = 2;", "a = 1; b = 2\n"},
	{"repeated_semicolons", "a();;b();", "a();b()\n"},
	{"compound_after_semicolon", "x = 1; if y {\n    z();\n}", "x = 1\nif y:\n  z()\n"},
	{"semicolon_inside_string", "print(\"a;\");\nx = 'b;'", "print(\"a;\")\nx = 'b;'\n"},
	{"string_ending_with_semicolon_no_terminator", "s = \";\"", "s = \";\"\n"},
	{"semicolon_before_trailing_comment", "x = 1; # c", "x = 1 # c\n"},
	{"space_before_trailing_semicolon", "x = 1 ;", "x = 1\n"},
	{"close_brace_followed_by_code", "if x {\n    a();\n} b();", "if x:\n  a()\nb()\n"},
	{"code_followed_by_close_brace", "if x {\n    a() }\nb();", "if x:\n  a()\nb()\n"},
	{"statement_semicolon_then_close_brace", "if x {\n    d = {1: 2}; }\ny();", "if x:\n  d = {1: 2}\ny()\n"},
	{"double_close_on_one_line", "if a {\n    if b {\n        c();\n}}\nd();", "if a:\n  if b:\n    c()\nd()\n"},
	{"close_brace_with_trailing_comment_quirk", "for a in b {\n    if c {\n        d()\n        } # note\n    e()\n    }", "for a in b:\n  if c:\n    d()\n  # note\n  e()\n"},
	{"header_trailing_comment", "if x {\n    a();\n} else { # c\n    b();\n}", "if x:\n  a()\nelse:\n  # c\n  b()\n"},
	{"header_comment_containing_braces", "if x { # {a}\n    y();\n}\nz();", "if x:\n  # {a}\n  y()\nz()\n"},
	{"comment_with_brace_on_header", "if x { # open {\n    pass;\n}", "if x:\n  # open {\n  pass\n"},
	{"comment_line_containing_close_brace", "def f() {\n    # }\n    return 1;\n}", "def f():\n  # }\n  return 1\n"},
	{"trailing_comment_with_colon_brace", "x = 1  # note: {\nif y {\n    pass;\n}", "x = 1  # note: {\nif y:\n  pass\n"},
	{"allman_braces", "if x\n{\n    a();\n}\nelse\n{\n    b();\n}", "if x:\n  a()\nelse:\n  b()\n"},
	{"allman_comment_gap", "if x # c\n{\n    a();\n}", "if x:\n  # c\n  a()\n"},
	{"leading_trivia_before_elif", "if a {\n    x();\n}\n# note\n\nelif b {\n    y();\n}", "if a:\n  x()\n# note\n\nelif b:\n  y()\n"},
	{"multiline_dict_with_brace_in_string", "d = {\n  \"a\": \"{\"\n};\nif x {\n    pass;\n}", "d = {\n  \"a\": \"{\"\n}\nif x:\n  pass\n"},
	{"multiline_dict_with_brace_in_comment", "d = {\n  \"a\": 1  # }\n};\nif x {\n    pass;\n}", "d = {\n  \"a\": 1  # }\n}\nif x:\n  pass\n"},
	{"multiline_dict_inside_block_relative_indent", "def f() {\n    d = {\n        \"a\": 1\n    };\n    return d;\n}", "def f():\n  d = {\n      \"a\": 1\n  }\n  return d\n"},
	{"multiline_dict_then_nested_def", "def f() {\n    d = {\n        1: 2\n    }\n    def g() {\n        pass\n    }\n}", "def f():\n  d = {\n      1: 2\n  }\n  def g():\n    pass\n"},
	{"multiline_dict_close_followed_by_code", "d = {\n  1: 2\n}; x = 3;\ny = 4;", "d = {\n  1: 2\n}; x = 3\ny = 4\n"},
	{"multiline_dict_tab_indented", "d = {\n\t\"a\": 1\n};", "d = {\n  \"a\": 1\n}\n"},
	{"multiline_call_arguments", "result = f(\n    a,\n    b\n);\nif x {\n    pass;\n}", "result = f(\n    a,\n    b\n)\nif x:\n  pass\n"},
	{"multiline_call_with_dict_argument", "f(\n    {\"a\": 1},\n    2\n);\nx = 1;", "f(\n    {\"a\": 1},\n    2\n)\nx = 1\n"},
	{"multiline_def_header", "def f(\n    a,\n    b\n) {\n    pass;\n}", "def f(\n    a,\n    b\n):\n  pass\n"},
	{"parenthesised_with_items", "with (\n    open(a) as f,\n    open(b) as g\n) {\n    pass;\n}", "with (\n    open(a) as f,\n    open(b) as g\n):\n  pass\n"},
	{"backslash_continuation_header", "if a and \\\n    b {\n    pass;\n}", "if a and \\\n    b:\n  pass\n"},
	{"implicit_concatenation_continuation", "s = (\"a\"\n     \"b\");", "s = (\"a\"\n    \"b\")\n"},
	{"triple_quoted_string_with_braces_and_def", "def f() {\n    s = \"\"\"\n    def g() {\n        }\n    \"\"\";\n    return s;\n}", "def f():\n  s = \"\"\"\n    def g() {\n        }\n    \"\"\"\n  return s\n"},
	{"triple_quoted_string_preserves_indentation", "def f() {\n    s = \"\"\"\nline1\n        line2\n\"\"\";\n    return s;\n}", "def f():\n  s = \"\"\"\nline1\n        line2\n\"\"\"\n  return s\n"},
	{"triple_quoted_string_line_is_close_brace", "if x {\n    s = '''\n}\n''';\n    y();\n}", "if x:\n  s = '''\n}\n'''\n  y()\n"},
	{"triple_quoted_string_hash_line", "s = \"\"\"\n# not a comment {\n\"\"\";\nx = 1;", "s = \"\"\"\n# not a comment {\n\"\"\"\nx = 1\n"},
	{"triple_quoted_crlf", "s = \"\"\"a\r\nb\"\"\";\r\n", "s = \"\"\"a\nb\"\"\"\n"},
	{"docstring_with_brace", "def f() {\n    \"\"\"Doc with { brace.\"\"\"\n    return 1;\n}", "def f():\n  \"\"\"Doc with { brace.\"\"\"\n  return 1\n"},
	{"raw_string_regex_quantifier", "if re.match(r\"\\d{3}\", s) {\n    pass;\n}", "if re.match(r\"\\d{3}\", s):\n  pass\n"},
	{"string_ending_in_escaped_backslash", "if s == \"\\\\\" {\n    pass;\n}", "if s == \"\\\\\":\n  pass\n"},
	{"escaped_quote_then_brace_in_string", "s = \"a\\\"{\";\nif x {\n    pass;\n}", "s = \"a\\\"{\"\nif x:\n  pass\n"},
	{"byte_string_with_brace", "if b\"{\" in data {\n    pass;\n}", "if b\"{\" in data:\n  pass\n"},
	{"u_prefix_string_with_brace", "if u\"{\" {\n    pass;\n}", "if u\"{\":\n  pass\n"},
	{"fstring_nested_format_spec", "print(f\"{x:{w}}\");\nif x {\n    pass;\n}", "print(f\"{x:{w}}\")\nif x:\n  pass\n"},
	{"fstring_conversion_and_nested_spec_in_header", "if f\"{x!r:>{w}}\" {\n    pass;\n}", "if f\"{x!r:>{w}}\":\n  pass\n"},
	{"fstring_nested_quotes_in_header", "if f\"{d['k']}\" {\n    pass;\n}", "if f\"{d['k']}\":\n  pass\n"},
	{"fstring_pep701_same_quotes", "if f\"{\"a\"}\" {\n    pass;\n}", "if f\"{\"a\"}\":\n  pass\n"},
	{"fstring_brace_inside_nested_string", "if f\"{'}'}\" == s {\n    pass;\n}", "if f\"{'}'}\" == s:\n  pass\n"},
	{"fstring_escaped_open_brace", "if f\"{{\" == s {\n    pass;\n}", "if f\"{{\" == s:\n  pass\n"},
	{"fstring_escaped_braces_balanced", "if f\"{{literal}}\" == s {\n    pass;\n}", "if f\"{{literal}}\" == s:\n  pass\n"},
	{"rf_prefix_fstring", "if rf\"{x}\\d\" {\n    pass;\n}", "if rf\"{x}\\d\":\n  pass\n"},
	{"fstring_with_dict_expression", "x = f\"{ {'a': 1}['a'] }\";\nif x {\n    pass;\n}", "x = f\"{ {'a': 1}['a'] }\"\nif x:\n  pass\n"},
	{"fstring_walrus_is_format_spec", "x = f\"{x:=5}\";", "x = f\"{x:=5}\"\n"},
	{"tstring_in_header", "if t\"{x}\" {\n    pass;\n}", "if t\"{x}\":\n  pass\n"},
	{"already_colon_header", "if x: {\n    pass;\n}", "if x:\n  pass\n"},
	{"crlf_line_endings", "if x {\r\n    pass;\r\n}\r\n", "if x:\n  pass\n"},
	{"comment_ends_at_lone_carriage_return", "if x {\n    # a\rb = 1\n    pass;\n}", "if x:\n  # a\n  b = 1\n  pass\n"},
	{"comment_then_code_on_same_line_in_brackets", "x = (1, #c\r2)", "x = (1, #c\n2)\n"},
	{"comment_then_line_join_in_brackets", "x = (1, #c\r\\\n    2)", "x = (1, #c\n    2)\n"},
	{"tab_indentation", "if x {\n\tpass;\n}", "if x:\n  pass\n"},
	{"utf8_bom", "\ufeffif x {\n    pass;\n}", "if x:\n  pass\n"},
	{"unicode_identifiers", "for ü in größe {\n    print(ü);\n}", "for ü in größe:\n  print(ü)\n"},
	{"leading_blank_lines", "\n\nx = 1;", "\n\nx = 1\n"},
	{"trailing_whitespace_line", "x = 1;\n   ", "x = 1\n\n"},
	{"empty_input", "", ""},
	{"global_statement", "def f() {\n    global x;\n    x = 1;\n}", "def f():\n  global x\n  x = 1\n"},
	{"keyword_prefixed_identifiers_as_simple_statements", "try_count = {};\nexcept_list = 1;\nwith_ = 1;\nwhile_ok = {1};", "try_count = {}\nexcept_list = 1\nwith_ = 1\nwhile_ok = {1}\n"},
	{"number_followed_by_keyword", "x = 1if y else 2;", "x = 1if y else 2\n"},
}

func TestEdgeCases(t *testing.T) {
	for _, c := range edgeCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			p := NewPythonPreprocessor(2)

			//when
			result, err := p.ProcessString(c.input)

			//then
			require.NoError(t, err)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestVeryLongLine(t *testing.T) {
	//given
	input := "x = \"" + strings.Repeat("a", 70000) + "\";"
	expected := "x = \"" + strings.Repeat("a", 70000) + "\"\n"
	p := NewPythonPreprocessor(2)

	//when
	result, err := p.ProcessString(input)

	//then
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestContinuationRescaleIndentFour(t *testing.T) {
	//given
	input := "def f() {\n  d = {\n    1: 2\n  };\n}"
	expected := "def f():\n    d = {\n        1: 2\n    }\n"
	p := NewPythonPreprocessor(4)

	//when
	result, err := p.ProcessString(input)

	//then
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
