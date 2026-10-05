package parser

import (
	"fmt"

	"go-Bython/internal/bython/ast"
	"go-Bython/internal/bython/diagnostic"
	"go-Bython/internal/bython/token"
)

type (
	exprContext uint8

	openGroup struct {
		kind             ast.GroupKind
		display          bool
		open             int
		mark             int
		elements         int
		lambdaPending    int
		firstHasKeyColon bool
		firstStarStar    bool
		sawFor           bool
	}

	exprState struct {
		stack      []openGroup
		refs       []*ast.Group
		want       bool
		prevString bool
		lambdaTop  int
		lastSig    int
	}
)

const (
	ctxStatement exprContext = iota
	ctxHeader
	ctxDecorator
)

const (
	maxBracketDepth = 200
)

var (
	statementOnly = [token.KwYield + 1]bool{
		token.KwDef:      true,
		token.KwClass:    true,
		token.KwTry:      true,
		token.KwWhile:    true,
		token.KwWith:     true,
		token.KwElif:     true,
		token.KwExcept:   true,
		token.KwFinally:  true,
		token.KwReturn:   true,
		token.KwImport:   true,
		token.KwPass:     true,
		token.KwBreak:    true,
		token.KwContinue: true,
		token.KwGlobal:   true,
		token.KwNonlocal: true,
		token.KwDel:      true,
		token.KwRaise:    true,
		token.KwAssert:   true,
	}
)

func (p *parser) expr(ctx exprContext) (ast.Expr, *diagnostic.SyntaxError) {
	toks := p.toks
	first := p.pos

	st := &p.ex
	*st = exprState{stack: st.stack[:0], refs: st.refs[:0], want: true, lastSig: -1}

	if ctx == ctxStatement && p.atTypeAlias() {
		p.pos += 2
		st.want = false
		st.lastSig = p.pos - 1
	}

	for {
		i := p.pos
		tok := &toks[i]
		depth := len(st.stack)

		switch tok.Kind {
		case token.KindName, token.KindNumber, token.KindEllipsis, token.KindString:
			if !st.want && !(st.prevString && tok.Kind == token.KindString) {
				return ast.Expr{}, p.bracketError(st.stack, diagnostic.ErrJuxtaposed, *tok, msgJuxtaposed)
			}
			st.want = false
		case token.KindKeyword:
			if err := p.keyword(st, *tok); err != nil {
				return ast.Expr{}, err
			}
		case token.KindOperator:
			if depth > 0 && st.stack[depth-1].display && st.lastSig == st.stack[depth-1].open && p.isDoubleStar(*tok) {
				st.stack[depth-1].firstStarStar = true
			}
			st.want = true
		case token.KindComma:
			if depth > 0 {
				st.stack[depth-1].elements++
			}
			st.want = true
		case token.KindColon:
			if depth == 0 && st.lambdaTop == 0 && ctx == ctxHeader {
				return p.finishExpr(first, st.refs), nil
			}
			st.colon()
		case token.KindLParen:
			if err := p.push(st, *tok, ast.GroupParen, ast.GroupCall, false); err != nil {
				return ast.Expr{}, err
			}
		case token.KindLBrack:
			if err := p.push(st, *tok, ast.GroupList, ast.GroupSubscript, false); err != nil {
				return ast.Expr{}, err
			}
		case token.KindLBrace:
			if !st.want && depth == 0 {
				return p.finishExpr(first, st.refs), nil
			}

			if !st.want {
				return ast.Expr{}, p.bracketError(st.stack, diagnostic.ErrBraceInBrackets, *tok, msgBraceInBrackets)
			}

			if err := p.push(st, *tok, ast.GroupDict, ast.GroupDict, true); err != nil {
				return ast.Expr{}, err
			}
		case token.KindRParen, token.KindRBrack, token.KindRBrace:
			if depth == 0 && tok.Kind == token.KindRBrace {
				return p.finishExpr(first, st.refs), nil
			}

			if err := p.pop(st, *tok); err != nil {
				return ast.Expr{}, err
			}
		case token.KindNewline, token.KindComment:
			if depth == 0 {
				return p.finishExpr(first, st.refs), nil
			}
			p.pos++
			continue
		case token.KindLineJoin:
			p.pos++
			continue
		case token.KindSemicolon:
			if depth == 0 {
				return p.finishExpr(first, st.refs), nil
			}
			return ast.Expr{}, p.bracketError(st.stack, diagnostic.ErrSemicolonInBrackets, *tok, msgSemicolonInBrackets)
		case token.KindEOF:
			if depth == 0 {
				return p.finishExpr(first, st.refs), nil
			}

			opener := toks[st.stack[depth-1].open]
			return ast.Expr{}, diagnostic.At(p.src, diagnostic.ErrUnclosedBracket, opener, fmt.Sprintf(msgUnclosedBracket, p.src.Text[opener.Start]))
		}

		st.prevString = tok.Kind == token.KindString
		st.lastSig = i
		p.pos++
	}
}

func (p *parser) keyword(st *exprState, tok token.Token) *diagnostic.SyntaxError {
	depth := len(st.stack)

	switch tok.Kw {
	case token.KwTrue, token.KwFalse, token.KwNoneValue:
		if !st.want {
			return p.bracketError(st.stack, diagnostic.ErrJuxtaposed, tok, msgJuxtaposed)
		}
		st.want = false
		return nil
	case token.KwPass, token.KwBreak, token.KwContinue:
		if depth == 0 {
			if !st.want {
				return p.bracketError(st.stack, diagnostic.ErrJuxtaposed, tok, msgJuxtaposed)
			}
			st.want = false
			return nil
		}
	}

	if depth > 0 && statementOnly[tok.Kw] {
		return p.bracketError(st.stack, diagnostic.ErrUnexpected, tok, fmt.Sprintf(msgUnexpected, p.describe(tok)))
	}

	switch tok.Kw {
	case token.KwLambda:
		if depth == 0 {
			st.lambdaTop++
		} else {
			st.stack[depth-1].lambdaPending++
		}
	case token.KwFor:
		if depth > 0 {
			st.stack[depth-1].sawFor = true
		}
	}

	st.want = true

	return nil
}

func (st *exprState) colon() {
	depth := len(st.stack)

	switch {
	case depth == 0:
		if st.lambdaTop > 0 {
			st.lambdaTop--
		}
	case st.stack[depth-1].lambdaPending > 0:
		st.stack[depth-1].lambdaPending--
	case st.stack[depth-1].display && st.stack[depth-1].elements == 0:
		st.stack[depth-1].firstHasKeyColon = true
	}

	st.want = true
}

func (p *parser) push(st *exprState, tok token.Token, afterOperator, afterOperand ast.GroupKind, display bool) *diagnostic.SyntaxError {
	if len(st.stack) >= maxBracketDepth {
		return diagnostic.At(p.src, diagnostic.ErrTooDeep, tok, fmt.Sprintf(msgTooManyBrackets, maxBracketDepth))
	}

	kind := afterOperand
	if st.want {
		kind = afterOperator
	}

	st.stack = append(st.stack, openGroup{kind: kind, display: display, open: p.pos, mark: len(st.refs)})
	st.want = true

	return nil
}

func (p *parser) pop(st *exprState, tok token.Token) *diagnostic.SyntaxError {
	depth := len(st.stack)
	if depth == 0 {
		return diagnostic.At(p.src, diagnostic.ErrUnmatchedClose, tok, fmt.Sprintf(diagnostic.MsgUnmatchedClose, p.src.Text[tok.Start]))
	}

	g := st.stack[depth-1]
	opener := p.toks[g.open]
	if token.CloserFor(opener.Kind) != tok.Kind {
		msg := fmt.Sprintf(msgMismatchedClose, p.src.Text[tok.Start], p.src.Text[opener.Start], opener.Line, diagnostic.Column(p.src, opener))
		return diagnostic.Related(p.src, diagnostic.ErrMismatchedClose, tok, opener, msg)
	}

	kind := g.kind
	if g.display {
		kind = classifyDisplay(g, st.lastSig)
	}

	group := p.groups.alloc()
	group.Kind = kind
	group.Open = g.open
	group.Close = p.pos
	group.Children = p.groupRefs.keep(st.refs[g.mark:])

	st.refs = append(st.refs[:g.mark], group)
	st.stack = st.stack[:depth-1]
	st.want = false

	return nil
}

func (p *parser) finishExpr(first int, refs []*ast.Group) ast.Expr {
	end := p.pos
	for end > first && p.toks[end-1].Kind == token.KindLineJoin {
		end--
	}

	return ast.Expr{Span: ast.Span{First: first, End: end}, Groups: p.groupRefs.keep(refs)}
}

func (p *parser) bracketError(stack []openGroup, code diagnostic.ErrorCode, tok token.Token, msg string) *diagnostic.SyntaxError {
	if len(stack) > 0 {
		opener := p.toks[stack[len(stack)-1].open]
		if opener.Line < tok.Line {
			return diagnostic.At(p.src, diagnostic.ErrUnclosedBracket, opener, fmt.Sprintf(msgUnclosedBracket, p.src.Text[opener.Start]))
		}
	}

	return diagnostic.At(p.src, code, tok, msg)
}

func (p *parser) atTypeAlias() bool {
	tok := p.toks[p.pos]
	if tok.Kind != token.KindName || p.src.Text[tok.Start:tok.End] != "type" {
		return false
	}

	return p.toks[p.pos+1].Kind == token.KindName
}

func (p *parser) isDoubleStar(tok token.Token) bool {
	return p.src.Text[tok.Start:tok.End] == "**"
}

func (p *parser) opensBlock(from int) bool {
	first := p.toks[from]
	if first.Kind == token.KindComma || (first.Kind == token.KindOperator && !p.isUnaryStart(first)) {
		return false
	}

	saved := p.pos
	p.pos = from
	ex, err := p.expr(ctxHeader)
	ok := err == nil && ex.Span.End > ex.Span.First && p.blockFollows(p.pos)
	p.pos = saved

	return ok
}

func (p *parser) isUnaryStart(tok token.Token) bool {
	switch p.src.Text[tok.Start:tok.End] {
	case "-", "+", "~", "*":
		return true
	}

	return false
}

func (p *parser) blockFollows(i int) bool {
	if p.toks[i].Kind == token.KindColon {
		i++
	}

	for p.toks[i].Kind == token.KindNewline || p.toks[i].Kind == token.KindComment {
		i++
	}

	return p.toks[i].Kind == token.KindLBrace
}

func classifyDisplay(g openGroup, lastSig int) ast.GroupKind {
	switch {
	case lastSig == g.open:
		return ast.GroupDict
	case g.sawFor && g.firstHasKeyColon:
		return ast.GroupDictComp
	case g.sawFor:
		return ast.GroupSetComp
	case g.firstHasKeyColon || g.firstStarStar:
		return ast.GroupDict
	}

	return ast.GroupSet
}
