package storage

import (
	"sort"

	"github.com/matt-in-space/tangodb/storage/record"
)

// MemoryStore is a Store held in memory. It keeps records as their encoded
// bytes, not as Entity values, so every write goes through record.Encode and
// every read through record.Decode, the same as a page-backed store will.
type MemoryStore struct {
	cfg Config

	records map[uint64][]byte // internal id -> encoded record
	keys    map[uint64]any    // internal id -> its primary key value
	index   map[any]uint64    // primary key value -> internal id
	auto    map[string]int64  // @auto field -> next value
	nextID  uint64            // starts at 1, so the zero ref never refers to a record
}

var _ Store = (*MemoryStore)(nil)

// NewMemoryStore creates an empty store for one collection.
func NewMemoryStore(cfg Config) (*MemoryStore, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	auto := make(map[string]int64, len(cfg.AutoFields))
	for _, field := range cfg.AutoFields {
		auto[field] = 1
	}

	return &MemoryStore{
		cfg:     cfg,
		records: map[uint64][]byte{},
		keys:    map[uint64]any{},
		index:   map[any]uint64{},
		auto:    auto,
		nextID:  1,
	}, nil
}

// Insert checks and encodes every record before storing any, so a failure
// stores nothing and leaves the @auto counters unchanged.
func (s *MemoryStore) Insert(records []record.Entity) ([]RecordRef, error) {
	type prepared struct {
		bytes []byte
		key   any
	}

	ready := make([]prepared, len(records))
	inBatch := map[any]bool{}

	// Prepare: check everything that could fail.
	for i, rec := range records {
		at := position(i, len(records))

		key, err := s.recordKey(rec, at)
		if err != nil {
			return nil, err
		}

		if s.cfg.PrimaryKey != "" {
			_, stored := s.index[key]
			if stored || inBatch[key] {
				return nil, errorf("%sduplicate primary key %s in collection %q", at, formatKey(key), s.cfg.Collection)
			}
			inBatch[key] = true
		}

		bytes, err := s.encode(rec, i, len(records))
		if err != nil {
			return nil, err
		}

		ready[i] = prepared{bytes: bytes, key: key}
	}

	// Apply: nothing below can fail.
	refs := make([]RecordRef, len(records))

	for i, p := range ready {
		id := s.nextID
		s.nextID++

		s.records[id] = p.bytes
		if s.cfg.PrimaryKey != "" {
			s.keys[id] = p.key
			s.index[p.key] = id
		}

		for _, field := range s.cfg.AutoFields {
			if value, ok := records[i][field].(int64); ok && value >= s.auto[field] {
				s.auto[field] = value + 1
			}
		}

		refs[i] = RecordRef{v: id}
	}

	return refs, nil
}

// Lookup finds a record by primary key.
func (s *MemoryStore) Lookup(key any) (RecordRef, record.Entity, bool, error) {
	if s.cfg.PrimaryKey == "" {
		return RecordRef{}, nil, false, errorf("collection %q has no primary key to look up by", s.cfg.Collection)
	}

	if err := s.checkKeyType(key, ""); err != nil {
		return RecordRef{}, nil, false, err
	}

	id, ok := s.index[key]
	if !ok {
		return RecordRef{}, nil, false, nil
	}

	rec, err := s.decode(id)
	if err != nil {
		return RecordRef{}, nil, false, err
	}

	return RecordRef{v: id}, rec, true, nil
}

// Update replaces a record, leaving it unchanged if anything fails.
func (s *MemoryStore) Update(ref RecordRef, rec record.Entity) error {
	if _, ok := s.records[ref.v]; !ok {
		return errorf("no record for this ref in collection %q", s.cfg.Collection)
	}

	if s.cfg.PrimaryKey != "" {
		key, err := s.recordKey(rec, "")
		if err != nil {
			return err
		}

		if old := s.keys[ref.v]; key != old {
			return errorf("update cannot change primary key %q from %s to %s in collection %q",
				s.cfg.PrimaryKey, formatKey(old), formatKey(key), s.cfg.Collection)
		}
	}

	bytes, err := s.encode(rec, 0, 1)
	if err != nil {
		return err
	}

	s.records[ref.v] = bytes
	return nil
}

// Delete removes a record and its index entry.
func (s *MemoryStore) Delete(ref RecordRef) error {
	if _, ok := s.records[ref.v]; !ok {
		return errorf("no record for this ref in collection %q", s.cfg.Collection)
	}

	delete(s.records, ref.v)
	if key, ok := s.keys[ref.v]; ok {
		delete(s.index, key)
		delete(s.keys, ref.v)
	}

	return nil
}

// NextAuto returns the next value of an @auto field's counter.
func (s *MemoryStore) NextAuto(field string) (int64, error) {
	next, ok := s.auto[field]
	if !ok {
		return 0, errorf("field %q is not an auto-increment field in collection %q", field, s.cfg.Collection)
	}
	return next, nil
}

// Scan visits the records that exist now, once each. It works from a
// snapshot of their ids, skipping any deleted before the cursor reaches
// them. The memory store happens to go in insertion order; callers mustn't
// rely on that.
func (s *MemoryStore) Scan() (Cursor, error) {
	ids := make([]uint64, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	return &memoryCursor{store: s, ids: ids, pos: -1}, nil
}

// recordKey checks a record's primary key (present, and the right type) and
// returns it, or nil when the collection has no primary key. at names the
// record within a batch, for messages.
func (s *MemoryStore) recordKey(rec record.Entity, at string) (any, error) {
	if s.cfg.PrimaryKey == "" {
		return nil, nil
	}

	key, ok := rec[s.cfg.PrimaryKey]
	if !ok || key == nil {
		return nil, errorf("%smissing primary key %q for collection %q", at, s.cfg.PrimaryKey, s.cfg.Collection)
	}

	if err := s.checkKeyType(key, at); err != nil {
		return nil, err
	}

	return key, nil
}

func (s *MemoryStore) checkKeyType(key any, at string) error {
	var ok bool
	switch s.cfg.KeyType {
	case IntKey:
		_, ok = key.(int64)
	case TextKey:
		_, ok = key.(string)
	}

	if !ok {
		return errorf("%sprimary key %q of collection %q is %s, got %s",
			at, s.cfg.PrimaryKey, s.cfg.Collection, s.cfg.KeyType, valueTypeName(key))
	}
	return nil
}

// encode turns a record into bytes and checks it fits in a page. i and count
// name the record within a batch, for messages.
func (s *MemoryStore) encode(rec record.Entity, i, count int) ([]byte, error) {
	bytes, err := record.Encode(rec)
	if err != nil {
		return nil, errorf("%scannot encode record: %v", position(i, count), err)
	}

	if len(bytes) > MaxRecordSize {
		if count > 1 {
			return nil, errorf("record %d is %d bytes; the maximum is %d", i+1, len(bytes), MaxRecordSize)
		}
		return nil, errorf("record is %d bytes; the maximum is %d", len(bytes), MaxRecordSize)
	}

	return bytes, nil
}

// decode turns a stored record back into a fresh, independent Entity.
func (s *MemoryStore) decode(id uint64) (record.Entity, error) {
	rec, err := record.Decode(s.records[id])
	if err != nil {
		return nil, errorf("cannot decode a record in collection %q: %v", s.cfg.Collection, err)
	}
	return rec, nil
}

type memoryCursor struct {
	store  *MemoryStore
	ids    []uint64
	pos    int
	rec    record.Entity
	err    error
	closed bool
}

func (c *memoryCursor) Next() bool {
	if c.closed || c.err != nil {
		return false
	}

	for c.pos+1 < len(c.ids) {
		c.pos++
		id := c.ids[c.pos]

		if _, ok := c.store.records[id]; !ok {
			continue // deleted since the scan started
		}

		rec, err := c.store.decode(id)
		if err != nil {
			c.err = err
			return false
		}

		c.rec = rec
		return true
	}

	c.rec = nil
	return false
}

func (c *memoryCursor) Ref() RecordRef {
	if c.pos < 0 || c.pos >= len(c.ids) {
		return RecordRef{}
	}
	return RecordRef{v: c.ids[c.pos]}
}

func (c *memoryCursor) Record() record.Entity { return c.rec }

func (c *memoryCursor) Err() error { return c.err }

func (c *memoryCursor) Close() error {
	c.closed = true
	c.rec = nil
	return nil
}
