package main

type DataType int

const (
	TypeInt DataType = iota
	TypeFloat
	TypeText
	TypeBool
)

type Collection struct {
	name       string
	data       map[string]DataType
	records    map[any]Entity
	primaryKey string
}

type Entity struct{}
