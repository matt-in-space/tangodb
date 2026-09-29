package main

import "fmt"

func ParseMerge(input string) (MergeOperation, error) {
	tokens, err := lex(input)
	if err != nil {
		return MergeOperation{}, err
	}

	p := &parser{tokens: tokens}

	op, err := p.parseMerge()
	if err != nil {
		return MergeOperation{}, err
	}

	return op.(MergeOperation), nil
}

func (p *parser) parseMerge() (Operation, error) {
	if err := p.expect(tokenMergeOp); err != nil {
		return nil, err
	}

	collectionName, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	// Filter parens are mandatory here, same as delete and for the same
	// reason: merge is a bulk mutating operation, so skipping the filter
	// position entirely shouldn't silently mean "match everything."
	if p.peek().kind == tokenEOF {
		return nil, ErrIncompleteInput
	}

	if p.peek().kind != tokenLParen {
		return nil, fmt.Errorf("merge requires an explicit filter, e.g. ~> %s() to match everything", collectionName)
	}

	filter, err := p.parseFilter()
	if err != nil {
		return nil, err
	}

	// The payload is mandatory too — parseRecordLiteral's own expect(LBrace)
	// naturally produces ErrIncompleteInput if we're still waiting on '{',
	// or a hard error if something else (like '=>') appears in its place.
	payload, err := p.parseRecordLiteral()
	if err != nil {
		return nil, err
	}

	var projection []string

	switch p.peek().kind {
	case tokenArrow:
		p.next()

		projection, err = p.parseProjection()
		if err != nil {
			return nil, err
		}

		// See parse_delete.go for why this stays non-nil even when empty:
		// it's the signal that '=>' was used at all, distinguishing
		// `=> {}` from omitting '=>' entirely.
		if projection == nil {
			projection = []string{}
		}

	case tokenEOF:
		return nil, ErrIncompleteInput
	}

	if err := p.expectEndOfStatement(); err != nil {
		return nil, err
	}

	return MergeOperation{
		Collection: collectionName,
		Filter:     filter,
		Payload:    payload,
		Projection: projection,
	}, nil
}
