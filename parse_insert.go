package main

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseInsert(input string) (InsertOperation, error) {
	tokens, err := lex(input)
	if err != nil {
		return InsertOperation{}, err
	}

	p := &parser{tokens: tokens}

	op, err := p.parseInsert()
	if err != nil {
		return InsertOperation{}, err
	}

	return op.(InsertOperation), nil
}

func (p *parser) parseInsert() (Operation, error) {
	if err := p.expect(tokenInsertOp); err != nil {
		return nil, err
	}

	collectionName, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	if err := p.expect(tokenArrow); err != nil {
		return nil, err
	}

	record, err := p.parseRecordLiteral()
	if err != nil {
		return nil, err
	}

	if err := p.expectEndOfStatement(); err != nil {
		return nil, err
	}

	return InsertOperation{
		Collection: collectionName,
		Record:     record,
	}, nil
}

func (p *parser) parseRecordLiteral() (Entity, error) {
	if err := p.expect(tokenLBrace); err != nil {
		return nil, err
	}

	record := Entity{}

	for p.peek().kind != tokenRBrace {
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

		record[fieldName] = value

		if p.peek().kind != tokenRBrace {
			if err := p.expect(tokenComma); err != nil {
				return nil, err
			}
		}
	}

	if err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}

	return record, nil
}

func (p *parser) parseValue() (any, error) {
	switch p.peek().kind {
	case tokenString:
		return p.next().value, nil

	case tokenNumber:
		text := p.next().value

		if strings.Contains(text, ".") {
			f, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q", text)
			}
			return f, nil
		}

		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", text)
		}
		return n, nil

	case tokenEOF:
		return nil, ErrIncompleteInput

	default:
		return nil, fmt.Errorf("expected a value, got %q", p.peek().value)
	}
}
