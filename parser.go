package main

import (
	"errors"
	"fmt"
)

// ErrIncompleteInput signals that parsing ran out of tokens at a point
// where more input could still make the statement valid (e.g. a REPL
// is mid-way through a multi-line statement). Callers that accumulate
// input across lines should keep reading on this error rather than
// treating it as a hard failure.
var ErrIncompleteInput = errors.New("incomplete input")

type parser struct {
	tokens []token
	pos    int
}

func Parse(input string) (Operation, error) {
	tokens, err := lex(input)
	if err != nil {
		return nil, err
	}

	p := &parser{tokens: tokens}

	switch {
	case p.peek().kind == tokenEOF:
		return nil, ErrIncompleteInput

	case p.peek().kind == tokenIdent && p.peekAt(1).kind == tokenLBrace:
		return p.parseDefineCollection()

	case p.peek().kind == tokenIdent && p.peekAt(1).kind == tokenEOF:
		return nil, ErrIncompleteInput

	default:
		return nil, fmt.Errorf("unrecognized statement")
	}
}

func (p *parser) peek() token {
	return p.tokens[p.pos]
}

func (p *parser) peekAt(offset int) token {
	i := p.pos + offset
	if i >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[i]
}

func (p *parser) next() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) expect(kind tokenKind) error {
	if p.peek().kind == tokenEOF {
		return ErrIncompleteInput
	}
	if p.peek().kind != kind {
		return fmt.Errorf("unexpected token %q", p.peek().value)
	}
	p.next()
	return nil
}

func (p *parser) expectIdent() (string, error) {
	if p.peek().kind == tokenEOF {
		return "", ErrIncompleteInput
	}
	if p.peek().kind != tokenIdent {
		return "", fmt.Errorf("expected identifier, got %q", p.peek().value)
	}
	return p.next().value, nil
}
