// Package database provides a SQL database engine with ACID transactions.
package database

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

// Table errors.
var (
	// ErrTableNotFound indicates the table was not found.
	ErrTableNotFound = errors.New("table not found")
	// ErrRowNotFound indicates the row was not found.
	ErrRowNotFound = errors.New("row not found")
	// ErrDuplicateRow indicates a duplicate row in a unique index.
	ErrDuplicateRow = errors.New("duplicate row")
	// ErrTableClosed indicates operations on a closed table.
	ErrTableClosed = errors.New("table is closed")
	// ErrInvalidRowData indicates invalid row data format.
	ErrInvalidRowData = errors.New("invalid row data")
)

// RowID represents a unique identifier for a row.
type RowID uint64

// InvalidRowID represents an invalid row ID.
const InvalidRowID RowID = 0

// Row represents a row in a table.
type Row struct {
	ID       RowID   // Unique row identifier
	SchemaID uint32  // Schema version when row was created
	Values   []Value // Column values
}

// NewRow creates a new row with the given values.
func NewRow(values []Value) *Row {
	return &Row{
		ID:     InvalidRowID,
		Values: values,
	}
}

// Serialize serializes the row to binary format.
func (r *Row) Serialize(schema *Schema) ([]byte, error) {
	if r == nil {
		return nil, ErrInvalidRowData
	}

	// Calculate size: 8 bytes for ID + 4 bytes for schema ID + variable for values
	size := 12 // ID (8) + SchemaID (4)
	for _, v := range r.Values {
		data, err := v.Serialize()
		if err != nil {
			return nil, fmt.Errorf("serialize value: %w", err)
		}
		size += 4 + len(data) // 4 bytes for length + data
	}

	buf := make([]byte, size)
	offset := 0

	// Write RowID
	binary.BigEndian.PutUint64(buf[offset:offset+8], uint64(r.ID))
	offset += 8

	// Write SchemaID
	binary.BigEndian.PutUint32(buf[offset:offset+4], r.SchemaID)
	offset += 4

	// Write values
	for _, v := range r.Values {
		data, err := v.Serialize()
		if err != nil {
			return nil, fmt.Errorf("serialize value: %w", err)
		}
		binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(len(data)))
		offset += 4
		copy(buf[offset:offset+len(data)], data)
		offset += len(data)
	}

	return buf, nil
}

// Deserialize deserializes a row from binary format.
func DeserializeRow(data []byte, schema *Schema) (*Row, error) {
	if len(data) < 12 {
		return nil, ErrInvalidRowData
	}

	offset := 0

	// Read RowID
	id := RowID(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Read SchemaID
	schemaID := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Read values
	values := make([]Value, len(schema.Columns))
	for i := 0; i < len(schema.Columns); i++ {
		if offset+4 > len(data) {
			return nil, ErrInvalidRowData
		}
		dataLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+dataLen > len(data) {
			return nil, ErrInvalidRowData
		}

		v, err := Deserialize(data[offset : offset+dataLen])
		if err != nil {
			return nil, fmt.Errorf("deserialize value: %w", err)
		}
		values[i] = v
		offset += dataLen
	}

	return &Row{
		ID:       id,
		SchemaID: schemaID,
		Values:   values,
	}, nil
}

// Table represents a table in the database.
type Table struct {
	name      string            // Table name
	schema    *Schema           // Table schema
	rows      map[RowID]*Row    // In-memory row cache
	nextRowID RowID             // Next row ID to assign
	indexes   map[string]*Index // Indexes on this table
	indexMgr  *IndexManager     // Index manager for this table
	mu        sync.RWMutex      // Table mutex
	closed    bool              // Whether table is closed
	rowCount  int64             // Number of rows
}

// NewTable creates a new table with the given schema.
func NewTable(name string, schema *Schema) (*Table, error) {
	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	idxMgr := NewIndexManager(name)
	table := &Table{
		name:      name,
		schema:    schema,
		rows:      make(map[RowID]*Row),
		nextRowID: 1,
		indexes:   make(map[string]*Index),
		indexMgr:  idxMgr,
		rowCount:  0,
	}

	// Create primary key index if primary key is defined
	if len(schema.PrimaryKey) > 0 {
		idxName := fmt.Sprintf("pk_%s", name)
		if err := idxMgr.CreateIndex(idxName, schema.PrimaryKey, true); err != nil {
			return nil, fmt.Errorf("create primary key index: %w", err)
		}
	}

	return table, nil
}

// Name returns the table name.
func (t *Table) Name() string {
	return t.name
}

// Schema returns the table schema.
func (t *Table) Schema() *Schema {
	return t.schema
}

// RowCount returns the number of rows in the table.
func (t *Table) RowCount() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.rowCount
}

// Insert inserts a new row into the table.
func (t *Table) Insert(values []Value) (RowID, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return InvalidRowID, ErrTableClosed
	}

	if len(values) != len(t.schema.Columns) {
		return InvalidRowID, fmt.Errorf("column count mismatch: got %d, want %d",
			len(values), len(t.schema.Columns))
	}

	// Validate NOT NULL constraints
	for i, col := range t.schema.Columns {
		if col.NotNull && values[i].IsNull() {
			return InvalidRowID, fmt.Errorf("column %s cannot be NULL", col.Name)
		}
	}

	row := &Row{
		ID:       t.nextRowID,
		SchemaID: 0, // Would be schema version in a real system
		Values:   values,
	}

	// Check unique constraints using primary key index
	if len(t.schema.PrimaryKey) > 0 {
		pkIdx, ok := t.indexMgr.GetIndex(fmt.Sprintf("pk_%s", t.name))
		if ok {
			pkKey := t.buildPrimaryKey(values)
			if _, err := pkIdx.Search(pkKey); err == nil {
				return InvalidRowID, ErrDuplicateRow
			}
			if err := pkIdx.Insert(pkKey, rowKey(t.nextRowID)); err != nil {
				return InvalidRowID, fmt.Errorf("insert primary key: %w", err)
			}
		}
	}

	t.rows[t.nextRowID] = row
	t.nextRowID++
	t.rowCount++

	return row.ID, nil
}

// buildPrimaryKey builds a primary key from the row values.
func (t *Table) buildPrimaryKey(values []Value) []byte {
	var key []byte
	for _, colName := range t.schema.PrimaryKey {
		idx := t.schema.GetColumnIndex(colName)
		if idx >= 0 {
			data, _ := values[idx].Serialize()
			key = append(key, data...)
		}
	}
	return key
}

// rowKey creates a key for storing row pointers in indexes.
func rowKey(id RowID) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(id))
	return b[:]
}

// Get retrieves a row by ID.
func (t *Table) Get(id RowID) (*Row, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, ErrTableClosed
	}

	row, ok := t.rows[id]
	if !ok {
		return nil, ErrRowNotFound
	}
	return row, nil
}

// Update updates an existing row.
func (t *Table) Update(id RowID, values []Value) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrTableClosed
	}

	if len(values) != len(t.schema.Columns) {
		return fmt.Errorf("column count mismatch: got %d, want %d",
			len(values), len(t.schema.Columns))
	}

	row, ok := t.rows[id]
	if !ok {
		return ErrRowNotFound
	}

	// Validate NOT NULL constraints
	for i, col := range t.schema.Columns {
		if col.NotNull && values[i].IsNull() {
			return fmt.Errorf("column %s cannot be NULL", col.Name)
		}
	}

	// If primary key changed, update indexes
	if len(t.schema.PrimaryKey) > 0 {
		oldPK := t.buildPrimaryKey(row.Values)
		newPK := t.buildPrimaryKey(values)
		if !bytes.Equal(oldPK, newPK) {
			pkIdx, ok := t.indexMgr.GetIndex(fmt.Sprintf("pk_%s", t.name))
			if ok {
				// Remove old key
				pkIdx.Delete(oldPK)
				// Add new key
				if _, err := pkIdx.Search(newPK); err == nil {
					return ErrDuplicateRow
				}
				if err := pkIdx.Insert(newPK, rowKey(id)); err != nil {
					return fmt.Errorf("update primary key: %w", err)
				}
			}
		}
	}

	row.Values = values
	return nil
}

// Delete deletes a row by ID.
func (t *Table) Delete(id RowID) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrTableClosed
	}

	row, ok := t.rows[id]
	if !ok {
		return ErrRowNotFound
	}

	// Remove from indexes
	if len(t.schema.PrimaryKey) > 0 {
		pkIdx, ok := t.indexMgr.GetIndex(fmt.Sprintf("pk_%s", t.name))
		if ok {
			pkKey := t.buildPrimaryKey(row.Values)
			pkIdx.Delete(pkKey)
		}
	}

	delete(t.rows, id)
	t.rowCount--
	return nil
}

// CreateIndex creates an index on the specified columns.
func (t *Table) CreateIndex(name string, columns []string, unique bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrTableClosed
	}

	// Validate columns exist
	for _, col := range columns {
		if !t.schema.HasColumn(col) {
			return fmt.Errorf("column %s not found", col)
		}
	}

	if err := t.indexMgr.CreateIndex(name, columns, unique); err != nil {
		return err
	}

	// Build index from existing rows
	idx, _ := t.indexMgr.GetIndex(name)
	for _, row := range t.rows {
		key := t.buildIndexKey(row.Values, columns)
		if err := idx.Insert(key, rowKey(row.ID)); err != nil {
			return fmt.Errorf("build index: %w", err)
		}
	}

	return nil
}

// buildIndexKey builds an index key from row values for the specified columns.
func (t *Table) buildIndexKey(values []Value, columns []string) []byte {
	var key []byte
	for _, col := range columns {
		idx := t.schema.GetColumnIndex(col)
		if idx >= 0 {
			data, _ := values[idx].Serialize()
			key = append(key, data...)
			key = append(key, 0xFF) // Separator
		}
	}
	return key
}

// DropIndex drops an index by name.
func (t *Table) DropIndex(name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrTableClosed
	}

	return t.indexMgr.DropIndex(name)
}

// GetIndex returns an index by name.
func (t *Table) GetIndex(name string) (*Index, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.indexMgr.GetIndex(name)
}

// Select returns all rows matching the given filter.
func (t *Table) Select(filter func(*Row) bool) ([]*Row, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, ErrTableClosed
	}

	var result []*Row
	for _, row := range t.rows {
		if filter == nil || filter(row) {
			result = append(result, row)
		}
	}
	return result, nil
}

// SelectByIndex performs an index lookup.
func (t *Table) SelectByIndex(indexName string, key []byte) ([]*Row, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, ErrTableClosed
	}

	idx, ok := t.indexMgr.GetIndex(indexName)
	if !ok {
		return nil, fmt.Errorf("index %s not found", indexName)
	}

	entries, err := idx.RangeQuery(key, append(key, 0xFF))
	if err != nil {
		return nil, err
	}

	var result []*Row
	for _, entry := range entries {
		rowID := RowID(binary.BigEndian.Uint64(entry.Value))
		if row, ok := t.rows[rowID]; ok {
			result = append(result, row)
		}
	}
	return result, nil
}

// Iterate iterates over all rows in the table.
func (t *Table) Iterate(fn func(*Row) error) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return ErrTableClosed
	}

	for _, row := range t.rows {
		if err := fn(row); err != nil {
			return err
		}
	}
	return nil
}

// Truncate removes all rows from the table.
func (t *Table) Truncate() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrTableClosed
	}

	t.rows = make(map[RowID]*Row)
	t.nextRowID = 1
	t.rowCount = 0

	// Clear all indexes
	if err := t.indexMgr.Close(); err != nil {
		return err
	}

	// Recreate primary key index
	if len(t.schema.PrimaryKey) > 0 {
		idxName := fmt.Sprintf("pk_%s", t.name)
		if err := t.indexMgr.CreateIndex(idxName, t.schema.PrimaryKey, true); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the table.
func (t *Table) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.closed = true
	t.rows = make(map[RowID]*Row)
	t.rowCount = 0

	if err := t.indexMgr.Close(); err != nil {
		return err
	}

	return nil
}

// TableManager manages multiple tables in a database.
type TableManager struct {
	tables map[string]*Table
	mu     sync.RWMutex
}

// NewTableManager creates a new table manager.
func NewTableManager() *TableManager {
	return &TableManager{
		tables: make(map[string]*Table),
	}
}

// CreateTable creates a new table.
func (m *TableManager) CreateTable(name string, schema *Schema) (*Table, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tables[name]; exists {
		return nil, fmt.Errorf("table %s already exists", name)
	}

	table, err := NewTable(name, schema)
	if err != nil {
		return nil, err
	}

	m.tables[name] = table
	return table, nil
}

// GetTable returns a table by name.
func (m *TableManager) GetTable(name string) (*Table, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	table, ok := m.tables[name]
	return table, ok
}

// DropTable drops a table by name.
func (m *TableManager) DropTable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	table, ok := m.tables[name]
	if !ok {
		return ErrTableNotFound
	}

	if err := table.Close(); err != nil {
		return err
	}

	delete(m.tables, name)
	return nil
}

// TableNames returns all table names.
func (m *TableManager) TableNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.tables))
	for name := range m.tables {
		names = append(names, name)
	}
	return names
}

// Close closes all tables.
func (m *TableManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, table := range m.tables {
		if err := table.Close(); err != nil {
			return err
		}
	}
	m.tables = make(map[string]*Table)
	return nil
}

// TableIterator iterates over rows in a table.
type TableIterator struct {
	table   *Table
	ids     []RowID
	current int
	mu      sync.RWMutex
}

// NewTableIterator creates a new table iterator.
func NewTableIterator(table *Table) *TableIterator {
	ids := make([]RowID, 0, table.rowCount)
	table.Iterate(func(row *Row) error {
		ids = append(ids, row.ID)
		return nil
	})
	return &TableIterator{
		table: table,
		ids:   ids,
	}
}

// Next returns the next row.
func (it *TableIterator) Next() (*Row, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.current >= len(it.ids) {
		return nil, io.EOF
	}

	row, err := it.table.Get(it.ids[it.current])
	if err != nil {
		return nil, err
	}
	it.current++
	return row, nil
}
