package core

import "fmt"

func ParseMerge(input string) (MergeOperation, error) {
	tokens, err := lexStatement(input)
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

	collectionName, err := p.expectPlainName("collection")
	if err != nil {
		return nil, err
	}

	// Filter parens are mandatory here, same as delete and for the same
	// reason: merge is a bulk mutating operation, so skipping the filter
	// position entirely shouldn't silently mean "match everything."

	if p.peek().kind != tokenLParen {
		return nil, fmt.Errorf("merge requires an explicit filter, e.g. ~> %s() to match everything", collectionName)
	}

	filter, err := p.parseFilter()
	if err != nil {
		return nil, err
	}

	// The payload is mandatory too — parseRecordLiteral's own expect(LBrace)
	// produces a hard error if something else (like '=>') appears in its place.
	p.inPayload = true
	payload, err := p.parseRecordLiteral("")
	p.inPayload = false
	if err != nil {
		return nil, err
	}

	payload, err = normalizePayload(payload)
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
