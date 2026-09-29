package core

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
	autoFields := map[string]bool{}
	optional := map[string]bool{}

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

		fieldIsID := false
		fieldIsAuto := false
		fieldIsOptional := false

		for p.peek().kind == tokenAt {
			p.next()

			annotation, err := p.expectIdent()
			if err != nil {
				return nil, err
			}

			switch annotation {
			case "id":
				if fieldIsID {
					return nil, fmt.Errorf("duplicate @id annotation on field %q", fieldName)
				}
				fieldIsID = true

			case "auto":
				if fieldIsAuto {
					return nil, fmt.Errorf("duplicate @auto annotation on field %q", fieldName)
				}
				fieldIsAuto = true

			case "optional":
				if fieldIsOptional {
					return nil, fmt.Errorf("duplicate @optional annotation on field %q", fieldName)
				}
				fieldIsOptional = true

			default:
				return nil, fmt.Errorf("unknown annotation %q on field %q", annotation, fieldName)
			}
		}

		if fieldIsOptional {
			if fieldIsID {
				return nil, fmt.Errorf("@id field %q cannot be @optional", fieldName)
			}
			if fieldIsAuto {
				return nil, fmt.Errorf("@auto field %q cannot be @optional", fieldName)
			}
			optional[fieldName] = true
		}

		if fieldIsID {
			if primaryKey != "" {
				return nil, fmt.Errorf("multiple @id fields declared (%q and %q)", primaryKey, fieldName)
			}
			primaryKey = fieldName
		}

		if fieldIsAuto {
			if dataType != TypeInt {
				return nil, fmt.Errorf("@auto requires an int field (field %q)", fieldName)
			}
			autoFields[fieldName] = true
		}
	}

	if err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}

	if err := p.expectEndOfStatement(); err != nil {
		return nil, err
	}

	return DefineCollectionOperation{
		Name:       name,
		Data:       data,
		PrimaryKey: primaryKey,
		AutoFields: autoFields,
		Optional:   optional,
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
