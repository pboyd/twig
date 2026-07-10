package filter

import (
	"fmt"
	"strings"
	"unicode"
)

// tokenType represents the type of a lexer token.
type tokenType int

const (
	tokenEOF tokenType = iota
	tokenIdent
	tokenString
	tokenInt
	tokenDate
	tokenAnd
	tokenHat
	tokenEq
	tokenNe
	tokenLt
	tokenLe
	tokenGt
	tokenGe
)

func (t tokenType) String() string {
	switch t {
	case tokenEOF:
		return "EOF"
	case tokenIdent:
		return "IDENT"
	case tokenString:
		return "STRING"
	case tokenInt:
		return "INT"
	case tokenDate:
		return "DATE"
	case tokenAnd:
		return "AND"
	case tokenHat:
		return "^"
	case tokenEq:
		return "="
	case tokenNe:
		return "!="
	case tokenLt:
		return "<"
	case tokenLe:
		return "<="
	case tokenGt:
		return ">"
	case tokenGe:
		return ">="
	}
	return "UNKNOWN"
}

type token struct {
	typ   tokenType
	value string
}

type lexer struct {
	input []rune
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: []rune(input)}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *lexer) advance() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r := l.input[l.pos]
	l.pos++
	return r
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *lexer) nextToken() (token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return token{typ: tokenEOF}, nil
	}

	ch := l.peek()

	// Quoted string
	if ch == '"' {
		return l.lexQuoted()
	}

	// Operators
	if ch == '!' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
		l.advance()
		l.advance()
		return token{typ: tokenNe, value: "!="}, nil
	}
	if ch == '=' {
		l.advance()
		return token{typ: tokenEq, value: "="}, nil
	}
	if ch == '<' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return token{typ: tokenLe, value: "<="}, nil
		}
		return token{typ: tokenLt, value: "<"}, nil
	}
	if ch == '>' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return token{typ: tokenGe, value: ">="}, nil
		}
		return token{typ: tokenGt, value: ">"}, nil
	}
	if ch == '^' {
		l.advance()
		return token{typ: tokenHat, value: "^"}, nil
	}

	// Digits may be a date (YYYY-MM-DD) or an integer.
	if ch >= '0' && ch <= '9' {
		return l.lexDateOrInt()
	}

	// Identifiers, keywords — start with letter
	if isIdentStart(ch) {
		return l.lexIdent()
	}

	return token{}, fmt.Errorf("unexpected character %q", string(ch))
}

func (l *lexer) lexQuoted() (token, error) {
	l.advance() // consume opening "
	var sb strings.Builder
	for {
		if l.pos >= len(l.input) {
			return token{}, fmt.Errorf("unterminated string: missing closing quote")
		}
		ch := l.advance()
		if ch == '"' {
			return token{typ: tokenString, value: sb.String()}, nil
		}
		sb.WriteRune(ch)
	}
}

func (l *lexer) lexInt() (token, error) {
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
		l.pos++
	}
	return token{typ: tokenInt, value: string(l.input[start:l.pos])}, nil
}

func (l *lexer) lexDateOrInt() (token, error) {
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
		l.pos++
	}
	word := string(l.input[start:l.pos])

	// Check if it looks like a date: YYYY-MM-DD
	// We've consumed the YYYY part; check for -MM-DD
	if l.pos+5 <= len(l.input) && l.input[l.pos] == '-' && l.input[l.pos+3] == '-' {
		// Try to consume -MM-DD
		saved := l.pos
		l.pos += 1 // skip first -
		if l.input[l.pos] >= '0' && l.input[l.pos] <= '9' && l.input[l.pos+1] >= '0' && l.input[l.pos+1] <= '9' {
			l.pos += 2
			if l.input[l.pos] == '-' {
				l.pos += 1
				if l.input[l.pos] >= '0' && l.input[l.pos] <= '9' && l.input[l.pos+1] >= '0' && l.input[l.pos+1] <= '9' {
					l.pos += 2
					full := string(l.input[start:l.pos])
					if len(full) == 10 && full[4] == '-' && full[7] == '-' &&
						isAllDigits(full[:4]) && isAllDigits(full[5:7]) && isAllDigits(full[8:10]) {
						return token{typ: tokenDate, value: full}, nil
					}
				}
			}
		}
		l.pos = saved // not a date, restore
	}

	return token{typ: tokenInt, value: word}, nil
}

func (l *lexer) lexIdent() (token, error) {
	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		l.pos++
	}
	word := string(l.input[start:l.pos])

	if strings.EqualFold(word, "AND") {
		return token{typ: tokenAnd, value: "AND"}, nil
	}

	return token{typ: tokenIdent, value: word}, nil
}

func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-'
}

func isAllDigits(s string) bool {
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
