package main

import (
	"fmt"
	"strings"
)

func ParseDefineCollection(input string) (DefineCollectionOperation, error) {
	tokens, err := lex(input)
	if err != nil {
		return DefineCollectionOperation{}, err
	}

	p := &parser{tokens: tokens}

	op, err := p.parseDefineCollection()
	if err != nil {
		return DefineCollectionOperation{}, err
	}

	return op.(DefineCollectionOperation), nil
}

func (p *parser) parseDefineCollection() (Operation, error) {
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	if err := p.expect(tokenLBrace); err != nil {
		return nil, err
	}

	data := map[string]DataType{}
	primaryKey := ""

	for p.peek().kind != tokenRBrace {
		fieldName, err := p.expectIdent()
		if err != nil {
			return nil, err
		}

		if err := p.expect(tokenColon); err != nil {
			return nil, err
		}

		typeName, err := p.expectIdent()
		if err != nil {
			return nil, err
		}

		dataType, err := parseDataType(typeName)
		if err != nil {
			return nil, err
		}

		data[fieldName] = dataType

		if p.peek().kind == tokenAt {
			p.next()

			annotation, err := p.expectIdent()
			if err != nil {
				return nil, err
			}

			if annotation != "id" {
				return nil, fmt.Errorf("unknown annotation %q on field %q", annotation, fieldName)
			}

			if primaryKey != "" {
				return nil, fmt.Errorf("multiple @id fields declared (%q and %q)", primaryKey, fieldName)
			}

			primaryKey = fieldName
		}
	}

	if err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}

	if p.peek().kind != tokenEOF {
		return nil, fmt.Errorf("unexpected input after collection definition: %q", p.peek().value)
	}

	return DefineCollectionOperation{
		Name:       name,
		Data:       data,
		PrimaryKey: primaryKey,
	}, nil
}

func parseDataType(name string) (DataType, error) {
	switch strings.ToLower(name) {
	case "int":
		return TypeInt, nil
	case "float":
		return TypeFloat, nil
	case "text":
		return TypeText, nil
	case "bool":
		return TypeBool, nil
	default:
		return 0, fmt.Errorf("unknown type %q", name)
	}
}
