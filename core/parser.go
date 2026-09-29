package core

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

	// A statement with an unclosed '{' or '(' isn't finished yet, so it isn't
	// parsed at all: a mistake partway through a multi-line statement is then
	// reported once, when the statement is complete, instead of the leftover
	// lines being read as new statements.
	if hasUnclosedBrackets(tokens) {
		return nil, ErrIncompleteInput
	}

	p := &parser{tokens: tokens}

	switch {
	case p.peek().kind == tokenEOF:
		return nil, ErrIncompleteInput

	case p.peek().kind == tokenInsertOp:
		return p.parseInsert()

	case p.peek().kind == tokenReadOp:
		return p.parseRead()

	case p.peek().kind == tokenDeleteOp:
		return p.parseDelete()

	case p.peek().kind == tokenMergeOp:
		return p.parseMerge()

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

// expectEndOfStatement tolerates one optional trailing ';' and then
// requires nothing but EOF. ';' is never required — a statement that's
// already structurally complete ends the same way with or without one —
// but some statements (like a bare read with no filter or projection)
// use it as an explicit "no more is coming" marker to resolve what would
// otherwise look like an incomplete statement.
func (p *parser) expectEndOfStatement() error {
	if p.peek().kind == tokenSemicolon {
		p.next()
	}

	if p.peek().kind != tokenEOF {
		return fmt.Errorf("unexpected input after statement: %q", p.peek().value)
	}

	return nil
}

// hasUnclosedBrackets reports whether tokens open more '{'/'(' than they close.
// If a closing bracket ever outnumbers its openers, more input can't fix the
// statement, so it counts as closed and the parser reports the error.
func hasUnclosedBrackets(tokens []token) bool {
	depth := 0

	for _, t := range tokens {
		switch t.kind {
		case tokenLBrace, tokenLParen:
			depth++
		case tokenRBrace, tokenRParen:
			depth--
			if depth < 0 {
				return false
			}
		}
	}

	return depth > 0
}
