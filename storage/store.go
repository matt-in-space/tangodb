// Package storage holds a collection's records for the query layer. A Store
// is the contract between the two: the query layer works with record.Entity
// values and knows the language's rules, and a Store keeps the records,
// finds them, and maintains the primary index and @auto counters.
//
// MemoryStore implements it in memory. A page-backed implementation (heap
// file plus B+tree) will implement the same interface later.
package storage

import (
	"fmt"

	"github.com/matt-in-space/tangodb/storage/record"
)

// The size limit comes from the page format, so the memory store and the
// page-backed store agree: a record has to fit in one data page, alongside
// the page header and its slot entry.
const (
	PageSize       = 4096
	PageHeaderSize = 16
	SlotSize       = 4
	MaxRecordSize  = PageSize - PageHeaderSize - SlotSize // 4076
)

// KeyType is the type of a collection's primary key.
type KeyType int

const (
	NoKey KeyType = iota
	IntKey
	TextKey
)

func (k KeyType) String() string {
	switch k {
	case IntKey:
		return "int"
	case TextKey:
		return "text"
	default:
		return "none"
	}
}

// Config is what a store needs to know about its collection. It's
// deliberately not the whole schema: validation stays in the query layer.
type Config struct {
	Collection string   // the collection's name, for messages
	PrimaryKey string   // the primary key field, or "" for none
	KeyType    KeyType  // IntKey or TextKey when PrimaryKey is set
	AutoFields []string // @auto fields; each counter starts at 1
}

func (c Config) validate() error {
	if c.PrimaryKey != "" && c.KeyType != IntKey && c.KeyType != TextKey {
		return errorf("primary key %q of collection %q must be int or text", c.PrimaryKey, c.Collection)
	}
	if c.PrimaryKey == "" && c.KeyType != NoKey {
		return errorf("collection %q has a key type but no primary key", c.Collection)
	}
	return nil
}

// RecordRef is an opaque handle to a stored record. Refs can be compared and
// used as map keys, but not built or inspected outside this package. The
// memory store keeps an internal id in it; the page-backed store will pack
// a RID (a 32-bit page and a 16-bit slot) into it. The zero ref never refers
// to a record.
type RecordRef struct{ v uint64 }

// Store holds one collection's records.
type Store interface {
	// Insert stores records all-or-nothing: every record is checked and
	// encoded before any is stored. It returns a ref for each, in input order.
	Insert(records []record.Entity) ([]RecordRef, error)

	// Lookup finds a record by primary key. A key that isn't stored reports
	// found = false, with no error.
	Lookup(key any) (ref RecordRef, rec record.Entity, found bool, err error)

	// Scan visits every record that exists when it starts, once each, in no
	// promised order. Callers that change records should collect refs first.
	Scan() (Cursor, error)

	// Update replaces a record. It can't change or remove the primary key,
	// and leaves the record unchanged if it fails.
	Update(ref RecordRef, rec record.Entity) error

	// Delete removes a record and its index entry.
	Delete(ref RecordRef) error

	// NextAuto returns the next value of an @auto field's counter. Counters
	// only advance when an insert succeeds.
	NextAuto(field string) (int64, error)
}

// Cursor walks the records a scan visits.
type Cursor interface {
	Next() bool            // advance; false at the end or on an error
	Ref() RecordRef        // the current record's ref
	Record() record.Entity // the current record: a decoded, independent copy
	Err() error            // the error that stopped the scan, if any
	Close() error
}

// errorf builds a storage error. Every storage error starts with "storage: "
// to set it apart from errors about the query language.
func errorf(format string, args ...any) error {
	return fmt.Errorf("storage: "+format, args...)
}

// position names a record within a batch, following the query layer's
// convention: "record N: " when there's more than one, nothing otherwise.
func position(i, count int) string {
	if count <= 1 {
		return ""
	}
	return fmt.Sprintf("record %d: ", i+1)
}

// formatKey shows a key in a message: text quoted, numbers as they are.
func formatKey(key any) string {
	if text, ok := key.(string); ok {
		return fmt.Sprintf("%q", text)
	}
	return fmt.Sprintf("%v", key)
}

// valueTypeName names a value's type in the query language's terms.
func valueTypeName(value any) string {
	switch value.(type) {
	case int64:
		return "int"
	case float64:
		return "float"
	case string:
		return "text"
	case bool:
		return "bool"
	case record.Entity:
		return "object"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("Go type %T", value)
	}
}
