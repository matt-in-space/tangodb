// Package record encodes records into TangoDB's own byte format and decodes
// them back. It sits between the query layer, which works with Entity values,
// and the storage layers below, which only ever see opaque bytes.
package record

// Entity is a record: field names to values. A value is an int64, float64,
// string, bool, or a nested Entity. A field with no value is absent, never
// nil.
type Entity map[string]any
