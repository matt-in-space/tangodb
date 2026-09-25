package main

type DataType int

const (
	TypeInt DataType = iota
	TypeFloat
	TypeText
	TypeBool
)

type Collection struct {
	name         string
	data         map[string]DataType
	records      map[uint64]Entity
	nextID       uint64
	primaryKey   string
	primaryIndex map[any]uint64
}

type Entity map[string]any
