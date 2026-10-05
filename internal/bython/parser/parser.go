package parser

import (
	"fmt"
	"slices"

	"go-Bython/internal/bython/ast"
	"go-Bython/internal/bython/diagnostic"
	"go-Bython/internal/bython/token"
)

type (
	headerRule uint8

	tryState uint8

	arena[T any] struct {
		chunk []T
	}

	Parser struct {
		p parser
	}

	parser struct {
		src         *token.Source
		toks        []token.Token
		pos         int
		blocks      int
		ex          exprState
		stmts       arena[ast.Stmt]
		simples     arena[ast.SimpleLine]
		simpleStmts arena[ast.SimpleStmt]
		compounds   arena[ast.CompoundStmt]
		clauses     arena[ast.Clause]
		comments    arena[ast.CommentStmt]
		groups      arena[ast.Group]
		groupRefs   arena[*ast.Group]
	}
)

const (
	headerNone headerRule = iota
	headerOptional
	headerRequired
	headerNamed
)

const (
	afterTry tryState = iota
	afterExcept
	afterTryElse
	afterFinally
)

const (
	maxBlockDepth = 100
	minChunk      = 32
)

const (
	msgUnmatchedBrace      = "unmatched '}': no block or bracket is open"
	msgMismatchedClose     = "closing '%c' does not match '%c' opened at %d:%d"
	msgUnclosedBracket     = "'%c' was never closed"
	msgUnclosedBlock       = "'{' was never closed (opens the '%s' block)"
	msgBlockAfterSimple    = "'{' cannot open a block here: only compound statements such as 'if', 'for', 'while', 'def' or 'class' take a block"
	msgBraceInBrackets     = "'{' cannot follow an expression inside brackets"
	msgSemicolonInBrackets = "';' is not allowed inside brackets"
	msgJuxtaposed          = "missing operator, ',' or ';' between expressions"
	msgUnexpected          = "unexpected %s"
	msgExpectedBlock       = "expected '{' to open the '%s' block"
	msgExpectedBlockColon  = " (Bython blocks use '{', not ':')"
	msgExpectedExpr        = "expected an expression after '%s'"
	msgExpectedName        = "expected a name after '%s'"
	msgBareKeyword         = "'%s' must be followed directly by '{'"
	msgBareKeywordElif     = "; use 'elif' for 'else if'"
	msgOrphanElse          = "'else' without a matching 'if', 'for', 'while' or 'try'"
	msgOrphanElif          = "'elif' without a matching 'if'"
	msgOrphanExcept        = "'except' without a matching 'try'"
	msgOrphanFinally       = "'finally' without a matching 'try'"
	msgClauseOrder         = "'%s' cannot follow '%s' at %d:%d"
	msgTryWithoutHandler   = "'try' block needs an 'except' or 'finally' block"
	msgMixedExcept         = "cannot mix 'except' and 'except*'"
	msgAsync               = "'async' must be followed by 'def', 'for' or 'with'"
	msgDecoratorTarget     = "a decorator must be followed by 'def', 'class' or 'async def'"
	msgCaseOutsideMatch    = "'case' block outside 'match'"
	msgNotCaseInMatch      = "only 'case' blocks may appear inside 'match'"
	msgMatchWithoutCase    = "'match' block needs at least one 'case' block"
	msgTooManyBlocks       = "too many nested blocks (limit %d)"
	msgTooManyBrackets     = "too many nested brackets (limit %d)"
)

var (
	ifFollowers      = []token.Keyword{token.KwElif, token.KwElse}
	elseOnly         = []token.Keyword{token.KwElse}
	tryFollowers     = []token.Keyword{token.KwExcept, token.KwFinally}
	exceptFollowers  = []token.Keyword{token.KwExcept, token.KwElse, token.KwFinally}
	tryElseFollowers = []token.Keyword{token.KwFinally}
	tryElseRejects   = []token.Keyword{token.KwExcept, token.KwElse}
	finallyRejects   = []token.Keyword{token.KwExcept, token.KwElse, token.KwFinally}

	blank = &ast.BlankStmt{}
)

func (a *arena[T]) alloc() *T {
	if len(a.chunk) == cap(a.chunk) {
		a.chunk = make([]T, 0, max(2*cap(a.chunk), minChunk))
	}

	a.chunk = a.chunk[:len(a.chunk)+1]

	return &a.chunk[len(a.chunk)-1]
}

func (a *arena[T]) keep(items []T) []T {
	if len(items) == 0 {
		return nil
	}

	if cap(a.chunk)-len(a.chunk) < len(items) {
		a.chunk = make([]T, 0, max(2*cap(a.chunk), len(items), minChunk))
	}

	start := len(a.chunk)
	a.chunk = append(a.chunk, items...)

	return a.chunk[start:len(a.chunk):len(a.chunk)]
}

func (a *arena[T]) reset() {
	clear(a.chunk)
	a.chunk = a.chunk[:0]
}

func Parse(src *token.Source) (*ast.Module, error) {
	var pp Parser
	m := &ast.Module{}

	if err := pp.Parse(m, src); err != nil {
		return nil, err
	}

	return m, nil
}

func (pp *Parser) Parse(m *ast.Module, src *token.Source) error {
	p := &pp.p
	p.src = src
	p.toks = src.Tokens
	p.pos = 0
	p.blocks = 0
	p.stmts.reset()
	p.simples.reset()
	p.simpleStmts.reset()
	p.compounds.reset()
	p.clauses.reset()
	p.comments.reset()
	p.groups.reset()
	p.groupRefs.reset()

	body, err := p.statements(false, nil)
	if err != nil {
		return err
	}

	m.Source = src
	m.Body = body

	return nil
}

func (p *parser) statements(inBlock bool, leading []ast.Stmt) ([]ast.Stmt, *diagnostic.SyntaxError) {
	var buf [16]ast.Stmt
	items := append(buf[:0], leading...)

	for {
		tok := p.toks[p.pos]

		switch tok.Kind {
		case token.KindEOF:
			return p.stmts.keep(items), nil
		case token.KindRBrace:
			if inBlock {
				return p.stmts.keep(items), nil
			}
			return nil, diagnostic.At(p.src, diagnostic.ErrUnmatchedClose, tok, msgUnmatchedBrace)
		case token.KindNewline:
			if tok.Flags&token.FlagBlank != 0 {
				items = append(items, blank)
			}
			p.pos++
		case token.KindSemicolon, token.KindLineJoin:
			p.pos++
		case token.KindComment:
			items = append(items, p.comment(p.pos))
			p.pos++
		default:
			st, err := p.statement()
			if err != nil {
				return nil, err
			}
			items = append(items, st)
		}
	}
}

func (p *parser) comment(i int) *ast.CommentStmt {
	c := p.comments.alloc()
	c.Tok = i

	return c
}

func (p *parser) statement() (ast.Stmt, *diagnostic.SyntaxError) {
	tok := p.toks[p.pos]

	switch tok.Kind {
	case token.KindKeyword:
		switch tok.Kw {
		case token.KwIf:
			return p.ifStmt()
		case token.KwWhile:
			return p.loopStmt(ast.CompoundWhile)
		case token.KwFor:
			return p.loopStmt(ast.CompoundFor)
		case token.KwTry:
			return p.tryStmt()
		case token.KwWith:
			return p.withStmt()
		case token.KwDef:
			return p.defStmt()
		case token.KwClass:
			return p.classStmt()
		case token.KwAsync:
			return p.asyncStmt()
		case token.KwElse, token.KwElif, token.KwExcept, token.KwFinally:
			return nil, p.orphan(tok)
		}
	case token.KindName:
		if p.isSoftKeyword(tok, "match") && p.opensBlock(p.pos+1) {
			return p.matchStmt()
		}

		if p.isSoftKeyword(tok, "case") && p.opensBlock(p.pos+1) {
			return nil, diagnostic.At(p.src, diagnostic.ErrCaseOutsideMatch, tok, msgCaseOutsideMatch)
		}
	case token.KindOperator:
		if p.isAt(tok) {
			return p.decorated()
		}
	}

	return p.simpleLine()
}

func (p *parser) orphan(tok token.Token) *diagnostic.SyntaxError {
	msg := msgOrphanElse
	switch tok.Kw {
	case token.KwElif:
		msg = msgOrphanElif
	case token.KwExcept:
		msg = msgOrphanExcept
	case token.KwFinally:
		msg = msgOrphanFinally
	}

	return diagnostic.At(p.src, diagnostic.ErrOrphanClause, tok, msg)
}

func (p *parser) isSoftKeyword(tok token.Token, word string) bool {
	return tok.Kind == token.KindName && p.src.Text[tok.Start:tok.End] == word
}

func (p *parser) isAt(tok token.Token) bool {
	return tok.Kind == token.KindOperator && p.src.Text[tok.Start:tok.End] == "@"
}

func (p *parser) isStar(tok token.Token) bool {
	return tok.Kind == token.KindOperator && p.src.Text[tok.Start:tok.End] == "*"
}

func (p *parser) startsCompound() bool {
	tok := p.toks[p.pos]

	switch tok.Kind {
	case token.KindKeyword:
		switch tok.Kw {
		case token.KwIf, token.KwWhile, token.KwFor, token.KwTry, token.KwWith, token.KwDef, token.KwClass, token.KwAsync, token.KwElif, token.KwElse, token.KwExcept, token.KwFinally:
			return true
		}
	case token.KindOperator:
		return p.isAt(tok)
	case token.KindName:
		return (p.isSoftKeyword(tok, "match") || p.isSoftKeyword(tok, "case")) && p.opensBlock(p.pos+1)
	}

	return false
}

func (p *parser) simpleLine() (*ast.SimpleLine, *diagnostic.SyntaxError) {
	var buf [4]ast.SimpleStmt
	stmts := buf[:0]

	for {
		ex, err := p.expr(ctxStatement)
		if err != nil {
			return nil, err
		}

		tok := p.toks[p.pos]
		if tok.Kind == token.KindLBrace {
			return nil, diagnostic.At(p.src, diagnostic.ErrBlockAfterSimple, tok, msgBlockAfterSimple)
		}

		if ex.Span.End == ex.Span.First {
			return nil, diagnostic.At(p.src, diagnostic.ErrUnexpected, tok, fmt.Sprintf(msgUnexpected, p.describe(tok)))
		}

		st := ast.SimpleStmt{Expr: ex, FirstSemi: -1, LastSemi: -1}
		if tok.Kind == token.KindSemicolon {
			st.FirstSemi = p.pos
			for p.toks[p.pos].Kind == token.KindSemicolon {
				st.LastSemi = p.pos
				p.pos++
			}
		}

		stmts = append(stmts, st)

		if st.FirstSemi < 0 || p.lineEnds() {
			break
		}
	}

	line := p.simples.alloc()
	line.Stmts = p.simpleStmts.keep(stmts)
	line.Trailing = -1

	tok := p.toks[p.pos]
	if tok.Kind == token.KindComment {
		line.Trailing = p.pos
		p.pos++
		tok = p.toks[p.pos]
	}

	switch tok.Kind {
	case token.KindNewline:
		p.pos++
	case token.KindRBrace, token.KindEOF:
	default:
		if line.Trailing >= 0 || !p.startsCompound() {
			return nil, diagnostic.At(p.src, diagnostic.ErrUnexpected, tok, fmt.Sprintf(msgUnexpected, p.describe(tok)))
		}
	}

	return line, nil
}

func (p *parser) lineEnds() bool {
	switch p.toks[p.pos].Kind {
	case token.KindNewline, token.KindComment, token.KindRBrace, token.KindEOF:
		return true
	}

	return p.startsCompound()
}

func (p *parser) decorated() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	var buf [2]ast.Decorator
	decorators := buf[:0]

	var leading []ast.Stmt
	for {
		at := p.pos
		atTok := p.toks[at]
		p.pos++

		ex, err := p.expr(ctxDecorator)
		if err != nil {
			return nil, err
		}

		tok := p.toks[p.pos]
		if ex.Span.End == ex.Span.First {
			return nil, diagnostic.After(p.src, diagnostic.ErrExpectedExpr, atTok, fmt.Sprintf(msgExpectedExpr, "@"))
		}

		if tok.Kind == token.KindLBrace {
			return nil, diagnostic.At(p.src, diagnostic.ErrBlockAfterSimple, tok, msgBlockAfterSimple)
		}

		d := ast.Decorator{Leading: leading, At: at, Expr: ex, Trailing: -1}
		if tok.Kind == token.KindComment {
			d.Trailing = p.pos
			p.pos++
			tok = p.toks[p.pos]
		}

		switch tok.Kind {
		case token.KindNewline:
			p.pos++
		case token.KindEOF:
		default:
			return nil, diagnostic.At(p.src, diagnostic.ErrUnexpected, tok, fmt.Sprintf(msgUnexpected, p.describe(tok)))
		}

		decorators = append(decorators, d)

		from := p.pos
		p.pos = p.skipTrivia(p.pos)
		leading = p.trivia(from, p.pos)

		if !p.isAt(p.toks[p.pos]) {
			break
		}
	}

	tok := p.toks[p.pos]
	next := p.toks[min(p.pos+1, len(p.toks)-1)]

	var cs *ast.CompoundStmt
	var err *diagnostic.SyntaxError
	switch {
	case tok.Kind == token.KindKeyword && tok.Kw == token.KwDef:
		cs, err = p.defStmt()
	case tok.Kind == token.KindKeyword && tok.Kw == token.KwClass:
		cs, err = p.classStmt()
	case tok.Kind == token.KindKeyword && tok.Kw == token.KwAsync && next.Kind == token.KindKeyword && next.Kw == token.KwDef:
		cs, err = p.asyncStmt()
	default:
		return nil, diagnostic.At(p.src, diagnostic.ErrDecoratorTarget, tok, msgDecoratorTarget)
	}
	if err != nil {
		return nil, err
	}

	cs.Decorators = append([]ast.Decorator(nil), decorators...)
	cs.Clauses[0].Leading = leading

	return cs, nil
}

func (p *parser) skipTrivia(i int) int {
	for {
		switch p.toks[i].Kind {
		case token.KindNewline, token.KindComment, token.KindLineJoin:
			i++
		default:
			return i
		}
	}
}

func (p *parser) trivia(from, to int) []ast.Stmt {
	var buf [8]ast.Stmt
	items := buf[:0]

	for i := from; i < to; i++ {
		tok := p.toks[i]
		switch {
		case tok.Kind == token.KindComment:
			items = append(items, p.comment(i))
		case tok.Kind == token.KindNewline && tok.Flags&token.FlagBlank != 0:
			items = append(items, blank)
		}
	}

	return p.stmts.keep(items)
}

func (p *parser) continuation(allowed, outOfOrder []token.Keyword, prev int) ([]ast.Stmt, bool, *diagnostic.SyntaxError) {
	i := p.skipTrivia(p.pos)
	tok := p.toks[i]
	if tok.Kind != token.KindKeyword {
		return nil, false, nil
	}

	if slices.Contains(allowed, tok.Kw) {
		leading := p.trivia(p.pos, i)
		p.pos = i
		return leading, true, nil
	}

	if slices.Contains(outOfOrder, tok.Kw) {
		prevTok := p.toks[prev]
		msg := fmt.Sprintf(msgClauseOrder, tok.Kw, p.src.Text[prevTok.Start:prevTok.End], prevTok.Line, diagnostic.Column(p.src, prevTok))
		return nil, false, diagnostic.Related(p.src, diagnostic.ErrClauseOrder, tok, prevTok, msg)
	}

	return nil, false, nil
}

func (p *parser) ifStmt() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	var buf [4]ast.Clause
	clauses := buf[:0]

	c, err := p.clause(ast.ClauseIf, headerRequired)
	if err != nil {
		return nil, err
	}
	clauses = append(clauses, c)

	for {
		leading, ok, err := p.continuation(ifFollowers, nil, c.Header.Span.First)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}

		kind, rule := ast.ClauseElif, headerRequired
		if p.toks[p.pos].Kw == token.KwElse {
			kind, rule = ast.ClauseElse, headerNone
		}

		c, err = p.clause(kind, rule)
		if err != nil {
			return nil, err
		}
		c.Leading = leading
		clauses = append(clauses, c)

		if kind == ast.ClauseElse {
			if _, _, err := p.continuation(nil, ifFollowers, c.Header.Span.First); err != nil {
				return nil, err
			}
			break
		}
	}

	return p.compound(ast.CompoundIf, clauses), nil
}

func (p *parser) loopStmt(kind ast.CompoundKind) (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	clauseKind := ast.ClauseWhile
	if kind == ast.CompoundFor {
		clauseKind = ast.ClauseFor
	}

	var buf [2]ast.Clause
	clauses := buf[:0]

	c, err := p.clause(clauseKind, headerRequired)
	if err != nil {
		return nil, err
	}
	clauses = append(clauses, c)

	leading, ok, err := p.continuation(elseOnly, nil, c.Header.Span.First)
	if err != nil {
		return nil, err
	}

	if ok {
		c, err = p.clause(ast.ClauseElse, headerNone)
		if err != nil {
			return nil, err
		}
		c.Leading = leading
		clauses = append(clauses, c)

		if _, _, err := p.continuation(nil, elseOnly, c.Header.Span.First); err != nil {
			return nil, err
		}
	}

	return p.compound(kind, clauses), nil
}

func (p *parser) tryStmt() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	tryTok := p.toks[p.pos]

	var buf [4]ast.Clause
	clauses := buf[:0]

	c, err := p.clause(ast.ClauseTry, headerNone)
	if err != nil {
		return nil, err
	}
	clauses = append(clauses, c)

	next := p.toks[p.skipTrivia(p.pos)]
	if next.Kind != token.KindKeyword || (next.Kw != token.KwExcept && next.Kw != token.KwFinally) {
		return nil, diagnostic.At(p.src, diagnostic.ErrTryWithoutHandler, tryTok, msgTryWithoutHandler)
	}

	state := afterTry
	star := -1
	for {
		allowed, outOfOrder := tryFollowers, []token.Keyword(nil)
		switch state {
		case afterExcept:
			allowed = exceptFollowers
		case afterTryElse:
			allowed, outOfOrder = tryElseFollowers, tryElseRejects
		case afterFinally:
			allowed, outOfOrder = nil, finallyRejects
		}

		leading, ok, err := p.continuation(allowed, outOfOrder, c.Header.Span.First)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}

		kwTok := p.toks[p.pos]
		switch kwTok.Kw {
		case token.KwExcept:
			kind := ast.ClauseExcept
			isStar := 0
			if p.isStar(p.toks[p.pos+1]) {
				kind = ast.ClauseExceptStar
				isStar = 1
			}

			if star >= 0 && star != isStar {
				return nil, diagnostic.At(p.src, diagnostic.ErrMixedExcept, kwTok, msgMixedExcept)
			}
			star = isStar

			c, err = p.clause(kind, headerOptional)
			state = afterExcept
		case token.KwElse:
			c, err = p.clause(ast.ClauseElse, headerNone)
			state = afterTryElse
		default:
			c, err = p.clause(ast.ClauseFinally, headerNone)
			state = afterFinally
		}
		if err != nil {
			return nil, err
		}

		c.Leading = leading
		clauses = append(clauses, c)
	}

	return p.compound(ast.CompoundTry, clauses), nil
}

func (p *parser) withStmt() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	c, err := p.clause(ast.ClauseWith, headerRequired)
	if err != nil {
		return nil, err
	}

	return p.compound(ast.CompoundWith, []ast.Clause{c}), nil
}

func (p *parser) defStmt() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	c, err := p.clause(ast.ClauseDef, headerNamed)
	if err != nil {
		return nil, err
	}

	return p.compound(ast.CompoundDef, []ast.Clause{c}), nil
}

func (p *parser) classStmt() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	c, err := p.clause(ast.ClauseClass, headerNamed)
	if err != nil {
		return nil, err
	}

	return p.compound(ast.CompoundClass, []ast.Clause{c}), nil
}

func (p *parser) matchStmt() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	c, err := p.clause(ast.ClauseMatch, headerRequired)
	if err != nil {
		return nil, err
	}

	return p.compound(ast.CompoundMatch, []ast.Clause{c}), nil
}

func (p *parser) caseStmt() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	c, err := p.clause(ast.ClauseCase, headerRequired)
	if err != nil {
		return nil, err
	}

	return p.compound(ast.CompoundCase, []ast.Clause{c}), nil
}

func (p *parser) asyncStmt() (*ast.CompoundStmt, *diagnostic.SyntaxError) {
	start := p.pos
	asyncTok := p.toks[start]
	next := p.toks[start+1]

	if next.Kind != token.KindKeyword || (next.Kw != token.KwDef && next.Kw != token.KwFor && next.Kw != token.KwWith) {
		return nil, diagnostic.At(p.src, diagnostic.ErrAsync, asyncTok, msgAsync)
	}

	p.pos++

	var cs *ast.CompoundStmt
	var err *diagnostic.SyntaxError
	switch next.Kw {
	case token.KwDef:
		cs, err = p.defStmt()
	case token.KwFor:
		cs, err = p.loopStmt(ast.CompoundFor)
	default:
		cs, err = p.withStmt()
	}
	if err != nil {
		return nil, err
	}

	cs.Clauses[0].Async = true
	cs.Clauses[0].Header.Span.First = start

	return cs, nil
}

func (p *parser) compound(kind ast.CompoundKind, clauses []ast.Clause) *ast.CompoundStmt {
	cs := p.compounds.alloc()
	cs.Kind = kind
	cs.Clauses = p.clauses.keep(clauses)

	return cs
}

func (p *parser) clause(kind ast.ClauseKind, rule headerRule) (ast.Clause, *diagnostic.SyntaxError) {
	keyword := p.pos

	header, pending, err := p.header(kind, rule)
	if err != nil {
		return ast.Clause{}, err
	}

	var block ast.Block
	if kind == ast.ClauseMatch {
		block, err = p.matchBlock(keyword, pending)
	} else {
		block, err = p.block(kind, keyword, pending)
	}
	if err != nil {
		return ast.Clause{}, err
	}

	return ast.Clause{Kind: kind, Header: header, Block: block}, nil
}

func (p *parser) header(kind ast.ClauseKind, rule headerRule) (ast.Header, []ast.Stmt, *diagnostic.SyntaxError) {
	first := p.pos
	kwTok := p.toks[first]
	p.pos++
	if kind == ast.ClauseExceptStar {
		p.pos++
	}

	h := ast.Header{Span: ast.Span{First: first, End: p.pos}, Colon: -1}
	next := p.toks[p.pos]

	switch rule {
	case headerNone:
		switch next.Kind {
		case token.KindColon, token.KindLBrace, token.KindNewline, token.KindComment, token.KindLineJoin:
		default:
			msg := fmt.Sprintf(msgBareKeyword, kind)
			if next.Kind == token.KindKeyword && next.Kw == token.KwIf {
				msg += msgBareKeywordElif
			}
			return ast.Header{}, nil, diagnostic.At(p.src, diagnostic.ErrBareKeyword, kwTok, msg)
		}
	case headerOptional:
		switch next.Kind {
		case token.KindColon, token.KindLBrace, token.KindNewline, token.KindComment:
		default:
			ex, err := p.expr(ctxHeader)
			if err != nil {
				return ast.Header{}, nil, err
			}
			h.Expr = ex
			h.Span.End = max(ex.Span.End, h.Span.End)
		}
	case headerRequired, headerNamed:
		if rule == headerNamed && next.Kind != token.KindName {
			return ast.Header{}, nil, diagnostic.After(p.src, diagnostic.ErrExpectedName, kwTok, fmt.Sprintf(msgExpectedName, kind))
		}

		ex, err := p.expr(ctxHeader)
		if err != nil {
			return ast.Header{}, nil, err
		}
		if ex.Span.End == ex.Span.First {
			return ast.Header{}, nil, diagnostic.After(p.src, diagnostic.ErrExpectedExpr, p.toks[h.Span.End-1], fmt.Sprintf(msgExpectedExpr, kind))
		}
		h.Expr = ex
		h.Span.End = ex.Span.End
	}

	if p.toks[p.pos].Kind == token.KindColon {
		h.Colon = p.pos
		p.pos++
	}

	var buf [4]ast.Stmt
	pending := buf[:0]
	for {
		tok := p.toks[p.pos]
		if tok.Kind == token.KindComment {
			pending = append(pending, p.comment(p.pos))
		} else if tok.Kind != token.KindNewline && tok.Kind != token.KindLineJoin {
			break
		}
		p.pos++
	}

	if p.toks[p.pos].Kind != token.KindLBrace {
		msg := fmt.Sprintf(msgExpectedBlock, kind)
		if h.Colon >= 0 {
			return ast.Header{}, nil, diagnostic.At(p.src, diagnostic.ErrExpectedBlock, p.toks[h.Colon], msg+msgExpectedBlockColon)
		}
		return ast.Header{}, nil, diagnostic.After(p.src, diagnostic.ErrExpectedBlock, p.toks[h.Span.End-1], msg)
	}

	return h, p.stmts.keep(pending), nil
}

func (p *parser) enterBlock() (int, *diagnostic.SyntaxError) {
	open := p.pos

	p.blocks++
	if p.blocks > maxBlockDepth {
		return 0, diagnostic.At(p.src, diagnostic.ErrTooDeep, p.toks[open], fmt.Sprintf(msgTooManyBlocks, maxBlockDepth))
	}

	p.pos++

	return open, nil
}

func (p *parser) block(kind ast.ClauseKind, keyword int, pending []ast.Stmt) (ast.Block, *diagnostic.SyntaxError) {
	open, err := p.enterBlock()
	if err != nil {
		return ast.Block{}, err
	}

	body, err := p.statements(true, pending)
	if err != nil {
		return ast.Block{}, err
	}

	if p.toks[p.pos].Kind != token.KindRBrace {
		return ast.Block{}, p.unclosedBlock(kind, open, keyword)
	}

	closeIdx := p.pos
	p.pos++
	p.blocks--

	return ast.Block{Open: open, Close: closeIdx, Body: body}, nil
}

func (p *parser) unclosedBlock(kind ast.ClauseKind, open, keyword int) *diagnostic.SyntaxError {
	name := kind.String()
	if kind == ast.ClauseExceptStar {
		name = ast.ClauseExcept.String()
	}

	return diagnostic.Related(p.src, diagnostic.ErrUnclosedBlock, p.toks[open], p.toks[keyword], fmt.Sprintf(msgUnclosedBlock, name))
}

func (p *parser) matchBlock(keyword int, pending []ast.Stmt) (ast.Block, *diagnostic.SyntaxError) {
	open, err := p.enterBlock()
	if err != nil {
		return ast.Block{}, err
	}

	var buf [16]ast.Stmt
	items := append(buf[:0], pending...)
	cases := 0

	for {
		tok := p.toks[p.pos]

		switch {
		case tok.Kind == token.KindEOF:
			return ast.Block{}, p.unclosedBlock(ast.ClauseMatch, open, keyword)
		case tok.Kind == token.KindRBrace:
			if cases == 0 {
				return ast.Block{}, diagnostic.At(p.src, diagnostic.ErrMatchWithoutCase, p.toks[keyword], msgMatchWithoutCase)
			}

			closeIdx := p.pos
			p.pos++
			p.blocks--
			return ast.Block{Open: open, Close: closeIdx, Body: p.stmts.keep(items)}, nil
		case tok.Kind == token.KindNewline:
			if tok.Flags&token.FlagBlank != 0 {
				items = append(items, blank)
			}
			p.pos++
		case tok.Kind == token.KindComment:
			items = append(items, p.comment(p.pos))
			p.pos++
		case tok.Kind == token.KindLineJoin || tok.Kind == token.KindSemicolon:
			p.pos++
		case p.isSoftKeyword(tok, "case"):
			cs, err := p.caseStmt()
			if err != nil {
				return ast.Block{}, err
			}
			items = append(items, cs)
			cases++
		default:
			return ast.Block{}, diagnostic.At(p.src, diagnostic.ErrNotCaseInMatch, tok, msgNotCaseInMatch)
		}
	}
}

func (p *parser) describe(tok token.Token) string {
	switch tok.Kind {
	case token.KindEOF:
		return "end of file"
	case token.KindNewline:
		return "newline"
	case token.KindComment:
		return "comment"
	case token.KindString:
		return "string literal"
	}

	return "'" + p.src.Text[tok.Start:tok.End] + "'"
}
