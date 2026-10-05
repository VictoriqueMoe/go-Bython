package token

import "strings"

type (
	Kind    uint8
	Keyword uint8
	Flags   uint8

	Token struct {
		Kind  Kind
		Kw    Keyword
		Flags Flags
		Start int32
		End   int32
		Line  int32
	}

	LineInfo struct {
		Start  int32
		Indent int32
	}

	Source struct {
		Text      string
		Tokens    []Token
		Lines     []LineInfo
		FirstLine int32
	}
)

const (
	KindEOF Kind = iota
	KindNewline
	KindLineJoin
	KindComment
	KindName
	KindKeyword
	KindNumber
	KindString
	KindEllipsis
	KindOperator
	KindLParen
	KindRParen
	KindLBrack
	KindRBrack
	KindLBrace
	KindRBrace
	KindComma
	KindColon
	KindSemicolon
)

const (
	NoKeyword Keyword = iota
	KwFalse
	KwNoneValue
	KwTrue
	KwAnd
	KwAs
	KwAssert
	KwAsync
	KwAwait
	KwBreak
	KwClass
	KwContinue
	KwDef
	KwDel
	KwElif
	KwElse
	KwExcept
	KwFinally
	KwFor
	KwFrom
	KwGlobal
	KwIf
	KwImport
	KwIn
	KwIs
	KwLambda
	KwNonlocal
	KwNot
	KwOr
	KwPass
	KwRaise
	KwReturn
	KwTry
	KwWhile
	KwWith
	KwYield
)

const (
	FlagBlank Flags = 1 << iota
	FlagFString
	FlagTriple
	FlagMultiline
)

const (
	maxKeywordLen   = 8
	maxPrefixLen    = 2
	ellipsis        = "..."
	operatorSeconds = "=*/<>"
)

var (
	keywordNames = [KwYield + 1]string{
		KwFalse:     "False",
		KwNoneValue: "None",
		KwTrue:      "True",
		KwAnd:       "and",
		KwAs:        "as",
		KwAssert:    "assert",
		KwAsync:     "async",
		KwAwait:     "await",
		KwBreak:     "break",
		KwClass:     "class",
		KwContinue:  "continue",
		KwDef:       "def",
		KwDel:       "del",
		KwElif:      "elif",
		KwElse:      "else",
		KwExcept:    "except",
		KwFinally:   "finally",
		KwFor:       "for",
		KwFrom:      "from",
		KwGlobal:    "global",
		KwIf:        "if",
		KwImport:    "import",
		KwIn:        "in",
		KwIs:        "is",
		KwLambda:    "lambda",
		KwNonlocal:  "nonlocal",
		KwNot:       "not",
		KwOr:        "or",
		KwPass:      "pass",
		KwRaise:     "raise",
		KwReturn:    "return",
		KwTry:       "try",
		KwWhile:     "while",
		KwWith:      "with",
		KwYield:     "yield",
	}

	keywordsByFirstByte = func() [256][]Keyword {
		var buckets [256][]Keyword
		for kw, name := range keywordNames {
			if name != "" {
				buckets[name[0]] = append(buckets[name[0]], Keyword(kw))
			}
		}

		return buckets
	}()

	stringPrefixes = map[string]bool{
		"r": true, "u": true, "b": true, "f": true, "t": true,
		"br": true, "rb": true, "fr": true, "rf": true, "tr": true, "rt": true,
	}

	threeByteOperators = [...]string{"**=", "//=", ">>=", "<<="}

	twoByteOperators = [...]string{
		"**", "//", "<<", ">>", "<=", ">=", "==", "!=", "->", ":=",
		"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "@=",
	}

	closers = [KindSemicolon + 1]Kind{
		KindLParen: KindRParen,
		KindLBrack: KindRBrack,
		KindLBrace: KindRBrace,
	}

	singleByteKinds = [256]Kind{
		'(': KindLParen,
		')': KindRParen,
		'[': KindLBrack,
		']': KindRBrack,
		'{': KindLBrace,
		'}': KindRBrace,
		',': KindComma,
		':': KindColon,
		';': KindSemicolon,
		'+': KindOperator,
		'-': KindOperator,
		'*': KindOperator,
		'/': KindOperator,
		'%': KindOperator,
		'@': KindOperator,
		'&': KindOperator,
		'|': KindOperator,
		'^': KindOperator,
		'~': KindOperator,
		'<': KindOperator,
		'>': KindOperator,
		'.': KindOperator,
		'=': KindOperator,
	}
)

func (s *Source) LineOf(tok Token) LineInfo {
	return s.Lines[tok.Line-s.FirstLine]
}

func (k Keyword) String() string {
	return keywordNames[k]
}

func CloserFor(opener Kind) Kind {
	return closers[opener]
}

func Lookup(name string) Keyword {
	if len(name) == 0 || len(name) > maxKeywordLen {
		return NoKeyword
	}

	for _, kw := range keywordsByFirstByte[name[0]] {
		if keywordNames[kw] == name {
			return kw
		}
	}

	return NoKeyword
}

func IsStringPrefix(name string) bool {
	if len(name) > maxPrefixLen {
		return false
	}

	var lower [maxPrefixLen]byte
	for i, c := range []byte(name) {
		lower[i] = c | 0x20
	}

	return stringPrefixes[string(lower[:len(name)])]
}

func LookupOperator(s string) (Kind, int) {
	if strings.HasPrefix(s, ellipsis) {
		return KindEllipsis, len(ellipsis)
	}

	if len(s) >= 2 && strings.IndexByte(operatorSeconds, s[1]) >= 0 {
		for _, op := range threeByteOperators {
			if strings.HasPrefix(s, op) {
				return KindOperator, len(op)
			}
		}

		for _, op := range twoByteOperators {
			if strings.HasPrefix(s, op) {
				return KindOperator, len(op)
			}
		}
	}

	kind := singleByteKinds[s[0]]
	if kind == KindEOF {
		return KindEOF, 0
	}

	return kind, 1
}
