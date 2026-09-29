package core

import (
	"fmt"
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokenIdent tokenKind = iota
	tokenColon
	tokenLBrace
	tokenRBrace
	tokenAt
	tokenComma
	tokenSemicolon
	tokenLParen
	tokenRParen
	tokenStar
	tokenInsertOp // >>
	tokenReadOp   // <<
	tokenDeleteOp // !>
	tokenMergeOp  // ~>
	tokenArrow    // =>
	tokenString
	tokenNumber
	tokenEOF
)

type token struct {
	kind  tokenKind
	value string
}

func lex(input string) ([]token, error) {
	var tokens []token
	runes := []rune(input)

	for i := 0; i < len(runes); {
		r := runes[i]

		switch {
		case unicode.IsSpace(r):
			i++

		case r == '{':
			tokens = append(tokens, token{kind: tokenLBrace, value: "{"})
			i++

		case r == '}':
			tokens = append(tokens, token{kind: tokenRBrace, value: "}"})
			i++

		case r == ':':
			tokens = append(tokens, token{kind: tokenColon, value: ":"})
			i++

		case r == ',':
			tokens = append(tokens, token{kind: tokenComma, value: ","})
			i++

		case r == ';':
			tokens = append(tokens, token{kind: tokenSemicolon, value: ";"})
			i++

		case r == '(':
			tokens = append(tokens, token{kind: tokenLParen, value: "("})
			i++

		case r == ')':
			tokens = append(tokens, token{kind: tokenRParen, value: ")"})
			i++

		case r == '*':
			tokens = append(tokens, token{kind: tokenStar, value: "*"})
			i++

		case r == '@':
			tokens = append(tokens, token{kind: tokenAt, value: "@"})
			i++

		case r == '>':
			if i+1 >= len(runes) {
				return nil, ErrIncompleteInput
			}
			if runes[i+1] != '>' {
				return nil, fmt.Errorf("unexpected character %q at position %d", r, i)
			}
			tokens = append(tokens, token{kind: tokenInsertOp, value: ">>"})
			i += 2

		case r == '<':
			if i+1 >= len(runes) {
				return nil, ErrIncompleteInput
			}
			if runes[i+1] != '<' {
				return nil, fmt.Errorf("unexpected character %q at position %d", r, i)
			}
			tokens = append(tokens, token{kind: tokenReadOp, value: "<<"})
			i += 2

		case r == '!':
			if i+1 >= len(runes) {
				return nil, ErrIncompleteInput
			}
			if runes[i+1] != '>' {
				return nil, fmt.Errorf("unexpected character %q at position %d", r, i)
			}
			tokens = append(tokens, token{kind: tokenDeleteOp, value: "!>"})
			i += 2

		case r == '~':
			if i+1 >= len(runes) {
				return nil, ErrIncompleteInput
			}
			if runes[i+1] != '>' {
				return nil, fmt.Errorf("unexpected character %q at position %d", r, i)
			}
			tokens = append(tokens, token{kind: tokenMergeOp, value: "~>"})
			i += 2

		case r == '=':
			if i+1 >= len(runes) {
				return nil, ErrIncompleteInput
			}
			if runes[i+1] != '>' {
				return nil, fmt.Errorf("unexpected character %q at position %d", r, i)
			}
			tokens = append(tokens, token{kind: tokenArrow, value: "=>"})
			i += 2

		case r == '"':
			value, next, err := lexString(runes, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{kind: tokenString, value: value})
			i = next

		case unicode.IsDigit(r):
			start := i
			i++
			for i < len(runes) && unicode.IsDigit(runes[i]) {
				i++
			}
			if i < len(runes) && runes[i] == '.' {
				i++
				for i < len(runes) && unicode.IsDigit(runes[i]) {
					i++
				}
			}
			tokens = append(tokens, token{kind: tokenNumber, value: string(runes[start:i])})

		case isIdentStartRune(r):
			start := i
			for i < len(runes) && isIdentRune(runes[i]) {
				i++
			}
			tokens = append(tokens, token{kind: tokenIdent, value: string(runes[start:i])})

		default:
			return nil, fmt.Errorf("unexpected character %q at position %d", r, i)
		}
	}

	tokens = append(tokens, token{kind: tokenEOF})

	return tokens, nil
}

// lexString scans a double-quoted string literal starting at the opening
// quote (runes[start] == '"'), supporting \" and \\ as escapes. It returns
// the unescaped value and the index just past the closing quote.
func lexString(runes []rune, start int) (string, int, error) {
	var sb strings.Builder
	i := start + 1

	for i < len(runes) {
		c := runes[i]

		if c == '\\' && i+1 < len(runes) && (runes[i+1] == '"' || runes[i+1] == '\\') {
			sb.WriteRune(runes[i+1])
			i += 2
			continue
		}

		if c == '"' {
			return sb.String(), i + 1, nil
		}

		sb.WriteRune(c)
		i++
	}

	return "", 0, ErrIncompleteInput
}

func isIdentStartRune(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
