package core

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseInsert(input string) (InsertOperation, error) {
	tokens, err := lexStatement(input)
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

	collectionName, err := p.expectPlainName("collection")
	if err != nil {
		return nil, err
	}

	record, err := p.parseRecordLiteral("")
	if err != nil {
		return nil, err
	}

	records := []Entity{record}

	for p.peek().kind == tokenAmp {
		p.next()

		record, err := p.parseRecordLiteral("")
		if err != nil {
			return nil, err
		}

		records = append(records, record)
	}

	var projection []string

	switch p.peek().kind {
	case tokenArrow:
		p.next()

		projection, err = p.parseProjection()
		if err != nil {
			return nil, err
		}

		// Non-nil means '=>' was used, so `=> {}` stays distinguishable
		// from no projection at all, same as delete and merge.
		if projection == nil {
			projection = []string{}
		}
	}

	if err := p.expectEndOfStatement(); err != nil {
		return nil, err
	}

	return InsertOperation{
		Collection: collectionName,
		Records:    records,
		Projection: projection,
	}, nil
}

// parseRecordLiteral parses a `{ field: value ... }` record. path is its
// dotted prefix ("" for a top-level record, "address." for the record inside
// address), used to name fields in errors.
func (p *parser) parseRecordLiteral(path string) (Entity, error) {
	if err := p.expect(tokenLBrace); err != nil {
		return nil, err
	}

	record := Entity{}

	for p.peek().kind != tokenRBrace {
		fieldName, err := p.expectPlainName("field")
		if err != nil {
			return nil, err
		}

		if _, repeated := record[fieldName]; repeated {
			return nil, fmt.Errorf("field %q is given more than once", path+fieldName)
		}

		if err := p.expect(tokenColon); err != nil {
			return nil, err
		}

		value, err := p.parseValue(path + fieldName + ".")
		if err != nil {
			return nil, err
		}

		record[fieldName] = value
	}

	if err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}

	return record, nil
}

// parseValue parses a value. path is the dotted prefix a nested record
// literal would have, for naming its fields in errors.
func (p *parser) parseValue(path string) (any, error) {
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

	case tokenIdent:
		switch p.peek().value {
		case "true":
			p.next()
			return true, nil
		case "false":
			p.next()
			return false, nil
		case "null":
			p.next()
			return nil, nil
		}
		return nil, fmt.Errorf("expected a value, got %q", p.peek().value)

	case tokenLBrace:
		if p.peekAt(1).kind == tokenStar {
			return p.parseWildcardObject()
		}

		// A nested record literal: the value of an embedded block field.
		record, err := p.parseRecordLiteral(path)
		if err != nil {
			return nil, err
		}
		return record, nil

	case tokenEOF:
		return nil, ErrIncompleteInput

	default:
		return nil, fmt.Errorf("expected a value, got %q", p.peek().value)
	}
}
