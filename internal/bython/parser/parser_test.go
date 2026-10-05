package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-Bython/internal/bython/ast"
	"go-Bython/internal/bython/lexer"
)

var (
	groupCases = []struct {
		name   string
		src    string
		open   int
		groups []ast.GroupKind
	}{
		{"dict_comprehension", "x = {k: v for k in x}", -1, []ast.GroupKind{ast.GroupDictComp}},
		{"dict_unpack", "x = {**a}", -1, []ast.GroupKind{ast.GroupDict}},
		{"empty_braces", "x = {}", -1, []ast.GroupKind{ast.GroupDict}},
		{"set_literal", "x = {1}", -1, []ast.GroupKind{ast.GroupSet}},
		{"set_comprehension", "x = {x for x in y}", -1, []ast.GroupKind{ast.GroupSetComp}},
		{"dict_literal", "x = {1: 2, 3: 4}", -1, []ast.GroupKind{ast.GroupDict}},
		{"dict_with_lambda_value", "x = {'a': lambda: 1}", -1, []ast.GroupKind{ast.GroupDict}},
		{"set_of_lambda", "x = {lambda: 1}", -1, []ast.GroupKind{ast.GroupSet}},
		{"set_with_slice", "x = {a[1:2]}", -1, []ast.GroupKind{ast.GroupSet, ast.GroupSubscript}},
		{"set_of_generator_call", "x = {f(y for y in z)}", -1, []ast.GroupKind{ast.GroupSet, ast.GroupCall}},
		{"call_then_subscript", "f(a)[0]", -1, []ast.GroupKind{ast.GroupCall, ast.GroupSubscript}},
		{"list_of_tuple", "x = [1, (2, 3)]", -1, []ast.GroupKind{ast.GroupList, ast.GroupParen}},
		{"dict_statement_subscript", "{1: 2}[1]", -1, []ast.GroupKind{ast.GroupDict, ast.GroupSubscript}},
		{"set_of_sets", "m = {{1,2},{3,4}}", -1, []ast.GroupKind{ast.GroupSet, ast.GroupSet, ast.GroupSet}},
		{"empty_dict_in_header", "if x == {} {\n}", 5, []ast.GroupKind{ast.GroupDict}},
		{"dict_inside_call_in_header", "if d.get('a', {}) {\n}", 10, []ast.GroupKind{ast.GroupCall, ast.GroupDict}},
		{"def_with_dict_default", "def f(x={}) {\n}", 8, []ast.GroupKind{ast.GroupCall, ast.GroupDict}},
		{"generic_def", "def f[T](x) {\n}", 8, []ast.GroupKind{ast.GroupSubscript, ast.GroupCall}},
		{"set_comp_in_for_header", "for k in {y for y in z} {\n}", 10, []ast.GroupKind{ast.GroupSetComp}},
		{"lambda_display_in_header", "if lambda: {} {\n}", 5, []ast.GroupKind{ast.GroupDict}},
		{"main_guard", "if __name__ == \"__main__\" {\n}", 4, nil},
		{"tolerated_colon", "if x: {\n}", 3, nil},
		{"allman", "if x\n{\n}", 3, nil},
		{"allman_with_comment", "if x # c\n{\n}", 4, nil},
		{"else_without_space", "if x {\n}else{\n}", 2, nil},
	}

	opensBlockCases = []struct {
		name string
		src  string
		want bool
	}{
		{"match_subject", "match x {\n}", true},
		{"match_tuple_subject", "match (a, b) {\n}", true},
		{"match_colon_brace", "match x: {\n}", true},
		{"match_allman", "match x\n{\n}", true},
		{"match_call", "match(x)", false},
		{"match_assignment", "match = 1", false},
		{"match_attribute", "match.group(1)", false},
		{"match_plain_python", "match x:\n    pass", false},
		{"case_wildcard", "case _ {\n}", true},
		{"case_mapping", "case {\"k\": v} {\n}", true},
	}
)

func TestGroupClassification(t *testing.T) {
	for _, c := range groupCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			src, err := lexer.Lex(c.src)
			require.NoError(t, err)

			//when
			m, err := Parse(src)

			//then
			require.NoError(t, err)
			require.NotEmpty(t, m.Body)

			var ex ast.Expr
			open := -1
			switch st := m.Body[0].(type) {
			case *ast.SimpleLine:
				ex = st.Stmts[0].Expr
			case *ast.CompoundStmt:
				ex = st.Clauses[0].Header.Expr
				open = st.Clauses[0].Block.Open
			default:
				t.Fatalf("unexpected first statement %T", st)
			}

			assert.Equal(t, c.open, open)
			assert.Equal(t, c.groups, flatten(ex.Groups))
		})
	}
}

func TestToleratedColonIsRecorded(t *testing.T) {
	//given
	src, err := lexer.Lex("if x: {\n}")
	require.NoError(t, err)

	//when
	m, err := Parse(src)

	//then
	require.NoError(t, err)
	header := m.Body[0].(*ast.CompoundStmt).Clauses[0].Header
	assert.Equal(t, 2, header.Colon)
	assert.Equal(t, ast.Span{First: 0, End: 2}, header.Span)
}

func TestOpensBlock(t *testing.T) {
	for _, c := range opensBlockCases {
		t.Run(c.name, func(t *testing.T) {
			//given
			src, err := lexer.Lex(c.src)
			require.NoError(t, err)
			p := parser{src: src, toks: src.Tokens}

			//when
			got := p.opensBlock(1)

			//then
			assert.Equal(t, c.want, got)
			assert.Equal(t, 0, p.pos)
		})
	}
}

func TestClauseChains(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		kinds []ast.ClauseKind
	}{
		{"if_elif_else", "if a {\n} elif b {\n} else {\n}", []ast.ClauseKind{ast.ClauseIf, ast.ClauseElif, ast.ClauseElse}},
		{"for_else", "for x in y {\n} else {\n}", []ast.ClauseKind{ast.ClauseFor, ast.ClauseElse}},
		{"while_else", "while x {\n} else {\n}", []ast.ClauseKind{ast.ClauseWhile, ast.ClauseElse}},
		{"try_full", "try {\n} except E {\n} except F {\n} else {\n} finally {\n}", []ast.ClauseKind{ast.ClauseTry, ast.ClauseExcept, ast.ClauseExcept, ast.ClauseElse, ast.ClauseFinally}},
		{"try_finally", "try {\n} finally {\n}", []ast.ClauseKind{ast.ClauseTry, ast.ClauseFinally}},
		{"except_star", "try {\n} except* E {\n}", []ast.ClauseKind{ast.ClauseTry, ast.ClauseExceptStar}},
		{"bare_except", "try {\n} except {\n}", []ast.ClauseKind{ast.ClauseTry, ast.ClauseExcept}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			//given
			src, err := lexer.Lex(c.src)
			require.NoError(t, err)

			//when
			m, err := Parse(src)

			//then
			require.NoError(t, err)
			require.Len(t, m.Body, 1)

			var kinds []ast.ClauseKind
			for _, cl := range m.Body[0].(*ast.CompoundStmt).Clauses {
				kinds = append(kinds, cl.Kind)
			}
			assert.Equal(t, c.kinds, kinds)
		})
	}
}

func TestChainLeadingTrivia(t *testing.T) {
	//given
	src, err := lexer.Lex("if a {\n}\n# note\n\nelif b {\n}")
	require.NoError(t, err)

	//when
	m, err := Parse(src)

	//then
	require.NoError(t, err)
	require.Len(t, m.Body, 1)
	leading := m.Body[0].(*ast.CompoundStmt).Clauses[1].Leading
	require.Len(t, leading, 2)
	assert.IsType(t, &ast.CommentStmt{}, leading[0])
	assert.IsType(t, &ast.BlankStmt{}, leading[1])
}

func TestChainEndsBeforeSiblingComment(t *testing.T) {
	//given
	src, err := lexer.Lex("if a {\n} # c\nx = 1")
	require.NoError(t, err)

	//when
	m, err := Parse(src)

	//then
	require.NoError(t, err)
	require.Len(t, m.Body, 3)
	assert.IsType(t, &ast.CompoundStmt{}, m.Body[0])
	assert.IsType(t, &ast.CommentStmt{}, m.Body[1])
	assert.IsType(t, &ast.SimpleLine{}, m.Body[2])
}

func TestAsyncHeaderSpan(t *testing.T) {
	//given
	src, err := lexer.Lex("async def f() {\n}")
	require.NoError(t, err)

	//when
	m, err := Parse(src)

	//then
	require.NoError(t, err)
	clause := m.Body[0].(*ast.CompoundStmt).Clauses[0]
	assert.True(t, clause.Async)
	assert.Equal(t, 0, clause.Header.Span.First)
	assert.Equal(t, ast.ClauseDef, clause.Kind)
}

func TestMatchStructure(t *testing.T) {
	//given
	src, err := lexer.Lex("match x {\n    # c\n    case 1 {\n    }\n\n    case _ {\n    }\n}")
	require.NoError(t, err)

	//when
	m, err := Parse(src)

	//then
	require.NoError(t, err)
	cs := m.Body[0].(*ast.CompoundStmt)
	assert.Equal(t, ast.CompoundMatch, cs.Kind)
	require.Len(t, cs.Clauses, 1)

	var kinds []string
	for _, st := range cs.Clauses[0].Block.Body {
		switch s := st.(type) {
		case *ast.CompoundStmt:
			assert.Equal(t, ast.CompoundCase, s.Kind)
			kinds = append(kinds, "case")
		case *ast.CommentStmt:
			kinds = append(kinds, "comment")
		case *ast.BlankStmt:
			kinds = append(kinds, "blank")
		}
	}
	assert.Equal(t, []string{"comment", "case", "blank", "case"}, kinds)
}

func TestSimpleLineSemicolons(t *testing.T) {
	//given
	src, err := lexer.Lex("a();;b() ; # c")
	require.NoError(t, err)

	//when
	m, err := Parse(src)

	//then
	require.NoError(t, err)
	line := m.Body[0].(*ast.SimpleLine)
	require.Len(t, line.Stmts, 2)
	assert.Equal(t, 3, line.Stmts[0].FirstSemi)
	assert.Equal(t, 4, line.Stmts[0].LastSemi)
	assert.Equal(t, 8, line.Stmts[1].FirstSemi)
	assert.Equal(t, 9, line.Trailing)
}

func flatten(groups []*ast.Group) []ast.GroupKind {
	var out []ast.GroupKind
	for _, g := range groups {
		out = append(out, g.Kind)
		out = append(out, flatten(g.Children)...)
	}

	return out
}
