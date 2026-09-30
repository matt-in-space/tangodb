package core

import (
	"errors"
	"fmt"
	"strings"
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
	tokens, err := lexStatement(input)
	if err != nil {
		return nil, err
	}

	p := &parser{tokens: tokens}

	switch {
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
// expectEndOfStatement requires the ';' that ends every statement, and then
// the end of input: one statement is accepted at a time.
func (p *parser) expectEndOfStatement() error {
	switch p.peek().kind {
	case tokenEOF:
		return ErrIncompleteInput
	case tokenSemicolon:
		p.next()
	default:
		return fmt.Errorf("unexpected input after statement: %q", p.peek().value)
	}

	if p.peek().kind != tokenEOF {
		return fmt.Errorf("unexpected input after statement: %q", p.peek().value)
	}

	return nil
}

// lexStatement lexes one statement and checks that it's complete. Every
// statement ends with ';', and until the first ';' outside any brackets
// arrives, the statement isn't finished, so it isn't parsed at all: a mistake
// partway through a multi-line statement is then reported once, when the
// statement is complete. Only one statement is accepted at a time. Because
// of this check, the parsers never reach the end of input mid-statement.
func lexStatement(input string) ([]token, error) {
	tokens, err := lex(input)
	if err != nil {
		return nil, err
	}

	end, found, unmatched := findStatementEnd(tokens)
	if unmatched {
		// Parse it anyway, so the stray bracket is reported right away.
		return tokens, nil
	}
	if !found {
		return nil, ErrIncompleteInput
	}
	if next := tokens[end+1]; next.kind != tokenEOF {
		return nil, fmt.Errorf("unexpected input after statement: %q", next.value)
	}

	return tokens, nil
}

// findStatementEnd returns the index of the first ';' outside any brackets,
// which is where a statement ends. A ';' inside a string is part of the
// string token, so it never counts. unmatched reports a closing bracket with
// no opener: more input can't fix that, so the caller parses right away and
// the parser reports the error.
func findStatementEnd(tokens []token) (end int, found bool, unmatched bool) {
	depth := 0

	for i, t := range tokens {
		switch t.kind {
		case tokenLBrace, tokenLParen:
			depth++
		case tokenRBrace, tokenRParen:
			depth--
			if depth < 0 {
				return 0, false, true
			}
		case tokenSemicolon:
			if depth == 0 {
				return i, true, false
			}
		}
	}

	return 0, false, false
}

// expectPlainName reads a name being declared or written, a collection name
// or a field name, where a dotted path (address.city) isn't allowed: a path
// names a field inside an embedded object, it isn't a name of its own.
func (p *parser) expectPlainName(what string) (string, error) {
	name, err := p.expectIdent()
	if err != nil {
		return "", err
	}

	if strings.Contains(name, ".") {
		return "", fmt.Errorf("%s name %q cannot contain \".\"", what, name)
	}

	return name, nil
}
