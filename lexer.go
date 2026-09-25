package main

import (
	"fmt"
	"unicode"
)

type tokenKind int

const (
	tokenIdent tokenKind = iota
	tokenColon
	tokenLBrace
	tokenRBrace
	tokenAt
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

		case r == '@':
			tokens = append(tokens, token{kind: tokenAt, value: "@"})
			i++

		case isIdentRune(r):
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

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
