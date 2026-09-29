package core

import "fmt"

func ParseDelete(input string) (DeleteOperation, error) {
	tokens, err := lex(input)
	if err != nil {
		return DeleteOperation{}, err
	}

	p := &parser{tokens: tokens}

	op, err := p.parseDelete()
	if err != nil {
		return DeleteOperation{}, err
	}

	return op.(DeleteOperation), nil
}

func (p *parser) parseDelete() (Operation, error) {
	if err := p.expect(tokenDeleteOp); err != nil {
		return nil, err
	}

	collectionName, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	// Unlike read, the filter parens are mandatory here — even empty
	// ones. EOF means more input could still supply them, so we keep
	// waiting; anything else in their place is a hard error, since
	// skipping the filter position entirely (not just leaving it empty)
	// is exactly the "forgot the WHERE clause" mistake this guards
	// against.
	if p.peek().kind == tokenEOF {
		return nil, ErrIncompleteInput
	}

	if p.peek().kind != tokenLParen {
		return nil, fmt.Errorf("delete requires an explicit filter, e.g. !> %s() to match everything", collectionName)
	}

	filter, err := p.parseFilter()
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

		// A non-nil (possibly empty) Projection is the signal that '=>'
		// was used at all — db.delete() relies on this to decide between
		// a count-only result and a RETURNING-style one, so `=> {}` must
		// stay distinguishable from omitting '=>' entirely even though
		// parseProjection() itself returns nil for an empty field list.
		if projection == nil {
			projection = []string{}
		}

	case tokenEOF:
		return nil, ErrIncompleteInput
	}

	if err := p.expectEndOfStatement(); err != nil {
		return nil, err
	}

	return DeleteOperation{
		Collection: collectionName,
		Filter:     filter,
		Projection: projection,
	}, nil
}
