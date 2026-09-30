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
	name, err := p.expectPlainName("collection")
	if err != nil {
		return nil, err
	}

	block, err := p.parseFieldBlock("")
	if err != nil {
		return nil, err
	}

	if err := p.expectEndOfStatement(); err != nil {
		return nil, err
	}

	return DefineCollectionOperation{
		Name:       name,
		Data:       block.schema.Data,
		PrimaryKey: block.primaryKey,
		AutoFields: block.autoFields,
		Optional:   block.schema.Optional,
		Objects:    block.schema.Objects,
	}, nil
}

// fieldBlock is a parsed `{ ... }` of field declarations. primaryKey and
// autoFields are only ever set at the top level.
type fieldBlock struct {
	schema     *Schema
	primaryKey string
	autoFields map[string]bool
}

// parseFieldBlock parses a `{ field: type @annotation ... }` block, where a
// field's type may itself be a block. path is the block's dotted prefix: ""
// at the top level, "address." inside an embedded block, and so on. It's
// used both to name fields in errors and to know whether @id and @auto are
// allowed, since an embedded block has no identity of its own.
func (p *parser) parseFieldBlock(path string) (fieldBlock, error) {
	if err := p.expect(tokenLBrace); err != nil {
		return fieldBlock{}, err
	}

	block := fieldBlock{
		schema: &Schema{
			Data:     map[string]DataType{},
			Optional: map[string]bool{},
			Objects:  map[string]*Schema{},
		},
		autoFields: map[string]bool{},
	}
	top := path == ""

	for p.peek().kind != tokenRBrace {
		fieldName, err := p.expectPlainName("field")
		if err != nil {
			return fieldBlock{}, err
		}
		fullName := path + fieldName

		if err := p.expect(tokenColon); err != nil {
			return fieldBlock{}, err
		}

		var dataType DataType

		if p.peek().kind == tokenLBrace {
			nested, err := p.parseFieldBlock(fullName + ".")
			if err != nil {
				return fieldBlock{}, err
			}
			dataType = TypeObject
			block.schema.Objects[fieldName] = nested.schema
		} else {
			typeName, err := p.expectIdent()
			if err != nil {
				return fieldBlock{}, err
			}

			dataType, err = parseDataType(typeName)
			if err != nil {
				return fieldBlock{}, err
			}
		}

		block.schema.Data[fieldName] = dataType

		fieldIsID := false
		fieldIsAuto := false
		fieldIsOptional := false

		for p.peek().kind == tokenAt {
			p.next()

			annotation, err := p.expectIdent()
			if err != nil {
				return fieldBlock{}, err
			}

			switch annotation {
			case "id":
				if fieldIsID {
					return fieldBlock{}, fmt.Errorf("duplicate @id annotation on field %q", fullName)
				}
				fieldIsID = true

			case "auto":
				if fieldIsAuto {
					return fieldBlock{}, fmt.Errorf("duplicate @auto annotation on field %q", fullName)
				}
				fieldIsAuto = true

			case "optional":
				if fieldIsOptional {
					return fieldBlock{}, fmt.Errorf("duplicate @optional annotation on field %q", fullName)
				}
				fieldIsOptional = true

			default:
				return fieldBlock{}, fmt.Errorf("unknown annotation %q on field %q", annotation, fullName)
			}
		}

		if !top {
			if fieldIsID {
				return fieldBlock{}, fmt.Errorf("@id is not allowed inside an embedded block (field %q)", fullName)
			}
			if fieldIsAuto {
				return fieldBlock{}, fmt.Errorf("@auto is not allowed inside an embedded block (field %q)", fullName)
			}
		}

		if dataType == TypeObject && (fieldIsID || fieldIsAuto) {
			return fieldBlock{}, fmt.Errorf("only @optional is allowed on an embedded block (field %q)", fullName)
		}

		if fieldIsOptional {
			if fieldIsID {
				return fieldBlock{}, fmt.Errorf("@id field %q cannot be @optional", fullName)
			}
			if fieldIsAuto {
				return fieldBlock{}, fmt.Errorf("@auto field %q cannot be @optional", fullName)
			}
			block.schema.Optional[fieldName] = true
		}

		if fieldIsID {
			if block.primaryKey != "" {
				return fieldBlock{}, fmt.Errorf("multiple @id fields declared (%q and %q)", block.primaryKey, fieldName)
			}
			block.primaryKey = fieldName
		}

		if fieldIsAuto {
			if dataType != TypeInt {
				return fieldBlock{}, fmt.Errorf("@auto requires an int field (field %q)", fieldName)
			}
			block.autoFields[fieldName] = true
		}
	}

	if err := p.expect(tokenRBrace); err != nil {
		return fieldBlock{}, err
	}

	return block, nil
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
