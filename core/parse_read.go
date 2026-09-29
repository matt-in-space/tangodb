package core

func ParseRead(input string) (ReadOperation, error) {
	tokens, err := lex(input)
	if err != nil {
		return ReadOperation{}, err
	}

	p := &parser{tokens: tokens}

	op, err := p.parseRead()
	if err != nil {
		return ReadOperation{}, err
	}

	return op.(ReadOperation), nil
}

func (p *parser) parseRead() (Operation, error) {
	if err := p.expect(tokenReadOp); err != nil {
		return nil, err
	}

	collectionName, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	filter := map[string]any{}

	if p.peek().kind == tokenLParen {
		filter, err = p.parseFilter()
		if err != nil {
			return nil, err
		}
	}

	var projection []string

	switch p.peek().kind {
	case tokenArrow:
		p.next()

		projection, err = p.parseProjection()
		if err != nil {
			return nil, err
		}

	case tokenEOF:
		return nil, ErrIncompleteInput
	}

	// Anything other than '=>' (handled above) or ';'/EOF (handled by
	// expectEndOfStatement) is caught there as an unexpected token —
	// e.g. `<< user)`. When neither '=>' nor EOF was seen, projection
	// stays nil, meaning "not specified" (db.read resolves that to a
	// count-only result, same as delete/merge).
	if err := p.expectEndOfStatement(); err != nil {
		return nil, err
	}

	return ReadOperation{
		Collection: collectionName,
		Filter:     filter,
		Projection: projection,
	}, nil
}

func (p *parser) parseFilter() (map[string]any, error) {
	if err := p.expect(tokenLParen); err != nil {
		return nil, err
	}

	filter := map[string]any{}

	for p.peek().kind != tokenRParen {
		fieldName, err := p.expectIdent()
		if err != nil {
			return nil, err
		}

		if err := p.expect(tokenColon); err != nil {
			return nil, err
		}

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		filter[fieldName] = value
	}

	if err := p.expect(tokenRParen); err != nil {
		return nil, err
	}

	return filter, nil
}

func (p *parser) parseProjection() ([]string, error) {
	if err := p.expect(tokenLBrace); err != nil {
		return nil, err
	}

	// A wildcard must be the sole content of the braces — `{*}` is the
	// whole projection. Anything else (`{id, *}` or `{*, id}`) is caught
	// by the ordinary logic: in the field-list loop below, expectIdent()
	// rejects '*' as not an identifier; here, expect(tokenRBrace) rejects
	// any field following '*' (commas are whitespace, so `{*, id}` reaches
	// this check as `* id`). No bespoke "can't mix" check is needed.
	if p.peek().kind == tokenStar {
		p.next()

		if err := p.expect(tokenRBrace); err != nil {
			return nil, err
		}

		return []string{"*"}, nil
	}

	var fields []string

	for p.peek().kind != tokenRBrace {
		fieldName, err := p.expectIdent()
		if err != nil {
			return nil, err
		}

		fields = append(fields, fieldName)
	}

	if err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}

	return fields, nil
}
