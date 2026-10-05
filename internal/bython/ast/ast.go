package ast

import "go-Bython/internal/bython/token"

type (
	Module struct {
		Source *token.Source
		Body   []Stmt
	}

	Stmt interface {
		stmt()
	}

	Span struct {
		First int
		End   int
	}

	GroupKind    uint8
	ClauseKind   uint8
	CompoundKind uint8

	Group struct {
		Kind     GroupKind
		Open     int
		Close    int
		Children []*Group
	}

	Expr struct {
		Span   Span
		Groups []*Group
	}

	SimpleStmt struct {
		Expr      Expr
		FirstSemi int
		LastSemi  int
	}

	SimpleLine struct {
		Stmts    []SimpleStmt
		Trailing int
	}

	Header struct {
		Span  Span
		Expr  Expr
		Colon int
	}

	Block struct {
		Open  int
		Close int
		Body  []Stmt
	}

	Clause struct {
		Kind    ClauseKind
		Async   bool
		Leading []Stmt
		Header  Header
		Block   Block
	}

	Decorator struct {
		Leading  []Stmt
		At       int
		Expr     Expr
		Trailing int
	}

	CompoundStmt struct {
		Kind       CompoundKind
		Decorators []Decorator
		Clauses    []Clause
	}

	CommentStmt struct {
		Tok int
	}

	BlankStmt struct{}
)

const (
	GroupParen GroupKind = iota
	GroupCall
	GroupList
	GroupSubscript
	GroupDict
	GroupSet
	GroupDictComp
	GroupSetComp
)

const (
	ClauseIf ClauseKind = iota
	ClauseElif
	ClauseElse
	ClauseWhile
	ClauseFor
	ClauseTry
	ClauseExcept
	ClauseExceptStar
	ClauseFinally
	ClauseWith
	ClauseDef
	ClauseClass
	ClauseMatch
	ClauseCase
)

const (
	CompoundIf CompoundKind = iota
	CompoundWhile
	CompoundFor
	CompoundTry
	CompoundWith
	CompoundDef
	CompoundClass
	CompoundMatch
	CompoundCase
)

var (
	clauseNames = [...]string{
		ClauseIf:         "if",
		ClauseElif:       "elif",
		ClauseElse:       "else",
		ClauseWhile:      "while",
		ClauseFor:        "for",
		ClauseTry:        "try",
		ClauseExcept:     "except",
		ClauseExceptStar: "except*",
		ClauseFinally:    "finally",
		ClauseWith:       "with",
		ClauseDef:        "def",
		ClauseClass:      "class",
		ClauseMatch:      "match",
		ClauseCase:       "case",
	}
)

func (*SimpleLine) stmt()   {}
func (*CompoundStmt) stmt() {}
func (*CommentStmt) stmt()  {}
func (*BlankStmt) stmt()    {}

func (k ClauseKind) String() string {
	return clauseNames[k]
}
