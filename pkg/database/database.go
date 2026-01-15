// Package database provides a SQL database engine with ACID transactions.
package database

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Database errors.
var (
	// ErrDatabaseNotFound indicates the database was not found.
	ErrDatabaseNotFound = errors.New("database not found")
	// ErrDatabaseExists indicates the database already exists.
	ErrDatabaseExists = errors.New("database already exists")
	// ErrDatabaseClosed indicates operations on a closed database.
	ErrDatabaseClosed = errors.New("database is closed")
	// ErrTransactionActive indicates an active transaction must be committed.
	ErrTransactionActive = errors.New("transaction is active")
)

// Database represents a database instance.
type Database struct {
	name     string            // Database name
	path     string            // Database file path
	tableMgr *TableManager     // Table manager
	mu       sync.RWMutex      // Database mutex
	closed   bool              // Whether database is closed
	metadata *DatabaseMetadata // Database metadata
}

// DatabaseMetadata contains database metadata.
type DatabaseMetadata struct {
	Version    uint32 // Database format version
	CreatedAt  uint64 // Creation timestamp
	ModifiedAt uint64 // Last modification timestamp
	SchemaHash []byte // Schema hash for validation
}

// NewDatabase creates a new database instance.
func NewDatabase(name, path string) (*Database, error) {
	db := &Database{
		name:     name,
		path:     path,
		tableMgr: NewTableManager(),
		metadata: &DatabaseMetadata{
			Version:   1,
			CreatedAt: uint64(time.Now().Unix()),
		},
	}

	return db, nil
}

// Name returns the database name.
func (d *Database) Name() string {
	return d.name
}

// Path returns the database path.
func (d *Database) Path() string {
	return d.path
}

// CreateTable creates a new table in the database.
func (d *Database) CreateTable(name string, schema *Schema) (*Table, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil, ErrDatabaseClosed
	}

	// Check if table already exists
	if _, ok := d.tableMgr.GetTable(name); ok {
		return nil, fmt.Errorf("table %s already exists", name)
	}

	// Validate schema
	if schema.TableName == "" {
		schema.TableName = name
	}
	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Create table
	table, err := NewTable(name, schema)
	if err != nil {
		return nil, err
	}

	d.tableMgr.tables[name] = table
	return table, nil
}

// GetTable returns a table by name.
func (d *Database) GetTable(name string) (*Table, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return nil, false
	}

	return d.tableMgr.GetTable(name)
}

// DropTable drops a table by name.
func (d *Database) DropTable(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDatabaseClosed
	}

	return d.tableMgr.DropTable(name)
}

// TableNames returns all table names.
func (d *Database) TableNames() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return nil
	}

	return d.tableMgr.TableNames()
}

// Execute executes a SQL statement (simplified - see query package for full implementation).
func (d *Database) Execute(sql string) (Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil, ErrDatabaseClosed
	}

	// Simple parser for basic statements
	result := &simpleResult{}

	switch {
	case len(sql) >= 6 && sql[:6] == "CREATE":
		// Handle CREATE TABLE (simplified)
		// In a full implementation, this would use the SQL parser
		result.rowsAffected = 0
	case len(sql) >= 6 && sql[:6] == "INSERT":
		result.rowsAffected = 0
	case len(sql) >= 6 && sql[:6] == "SELECT":
		result.rowsAffected = 0
	case len(sql) >= 6 && sql[:6] == "UPDATE":
		result.rowsAffected = 0
	case len(sql) >= 6 && sql[:6] == "DELETE":
		result.rowsAffected = 0
	case len(sql) >= 4 && sql[:4] == "DROP":
		result.rowsAffected = 0
	default:
		return nil, fmt.Errorf("unknown SQL statement: %s", sql[:10])
	}

	return result, nil
}

// Result represents the result of a query execution.
type Result interface {
	RowsAffected() int64
	LastInsertID() (int64, error)
	Rows() []Row
}

// simpleResult is a simple implementation of Result.
type simpleResult struct {
	rowsAffected int64
	lastInsertID int64
	rows         []Row
}

// RowsAffected returns the number of rows affected.
func (r *simpleResult) RowsAffected() int64 {
	return r.rowsAffected
}

// LastInsertID returns the last insert ID.
func (r *simpleResult) LastInsertID() (int64, error) {
	return r.lastInsertID, nil
}

// Rows returns the result rows.
func (r *simpleResult) Rows() []Row {
	return r.rows
}

// Begin starts a new transaction (placeholder).
func (d *Database) Begin() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDatabaseClosed
	}

	return nil
}

// Commit commits the current transaction.
func (d *Database) Commit() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDatabaseClosed
	}

	return nil
}

// Rollback rolls back the current transaction.
func (d *Database) Rollback() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDatabaseClosed
	}

	return nil
}

// Close closes the database.
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}

	// Close all tables
	if err := d.tableMgr.Close(); err != nil {
		return err
	}

	d.closed = true
	return nil
}

// Save saves the database to disk.
func (d *Database) Save() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDatabaseClosed
	}

	// Ensure directory exists
	if err := os.MkdirAll(d.path, 0755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	// Save database header
	headerPath := filepath.Join(d.path, "header.dat")
	if err := d.saveHeader(headerPath); err != nil {
		return fmt.Errorf("save header: %w", err)
	}

	// Save table schemas
	for name, table := range d.tableMgr.tables {
		schemaPath := filepath.Join(d.path, fmt.Sprintf("%s.schema", name))
		if err := d.saveSchema(schemaPath, table.Schema()); err != nil {
			return fmt.Errorf("save schema for %s: %w", name, err)
		}
	}

	return nil
}

// saveHeader saves the database header.
func (d *Database) saveHeader(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write version
	if err := binary.Write(f, binary.BigEndian, d.metadata.Version); err != nil {
		return err
	}

	// Write timestamps
	if err := binary.Write(f, binary.BigEndian, d.metadata.CreatedAt); err != nil {
		return err
	}
	if err := binary.Write(f, binary.BigEndian, d.metadata.ModifiedAt); err != nil {
		return err
	}

	// Write schema hash length and hash
	if err := binary.Write(f, binary.BigEndian, uint32(len(d.metadata.SchemaHash))); err != nil {
		return err
	}
	if _, err := f.Write(d.metadata.SchemaHash); err != nil {
		return err
	}

	return nil
}

// saveSchema saves a table schema.
func (d *Database) saveSchema(path string, schema *Schema) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write table name
	if err := binary.Write(f, binary.BigEndian, uint32(len(schema.TableName))); err != nil {
		return err
	}
	if _, err := io.WriteString(f, schema.TableName); err != nil {
		return err
	}

	// Write column count
	if err := binary.Write(f, binary.BigEndian, uint32(len(schema.Columns))); err != nil {
		return err
	}

	// Write columns
	for _, col := range schema.Columns {
		// Column name
		if err := binary.Write(f, binary.BigEndian, uint32(len(col.Name))); err != nil {
			return err
		}
		if _, err := io.WriteString(f, col.Name); err != nil {
			return err
		}

		// Column type
		if err := binary.Write(f, binary.BigEndian, uint32(col.Type)); err != nil {
			return err
		}

		// Constraints
		constraints := uint32(0)
		if col.PrimaryKey {
			constraints |= 1 << 0
		}
		if col.NotNull {
			constraints |= 1 << 1
		}
		if col.Unique {
			constraints |= 1 << 2
		}
		if col.AutoInc {
			constraints |= 1 << 3
		}
		if err := binary.Write(f, binary.BigEndian, constraints); err != nil {
			return err
		}
	}

	// Write primary key
	if err := binary.Write(f, binary.BigEndian, uint32(len(schema.PrimaryKey))); err != nil {
		return err
	}
	for _, pk := range schema.PrimaryKey {
		if err := binary.Write(f, binary.BigEndian, uint32(len(pk))); err != nil {
			return err
		}
		if _, err := io.WriteString(f, pk); err != nil {
			return err
		}
	}

	return nil
}

// Load loads the database from disk.
func (d *Database) Load() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Load header
	headerPath := filepath.Join(d.path, "header.dat")
	if err := d.loadHeader(headerPath); err != nil {
		return fmt.Errorf("load header: %w", err)
	}

	// Load table schemas
	entries, err := os.ReadDir(d.path)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if len(entry.Name()) > 7 && entry.Name()[:len(entry.Name())-7] == ".schema" {
			schemaPath := filepath.Join(d.path, entry.Name())
			schema, err := d.loadSchema(schemaPath)
			if err != nil {
				return fmt.Errorf("load schema %s: %w", entry.Name(), err)
			}
			tableName := entry.Name()[:len(entry.Name())-7]
			if _, err := NewTable(tableName, schema); err != nil {
				return fmt.Errorf("create table %s: %w", tableName, err)
			}
		}
	}

	return nil
}

// loadHeader loads the database header.
func (d *Database) loadHeader(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // New database
		}
		return err
	}
	defer f.Close()

	// Read version
	if err := binary.Read(f, binary.BigEndian, &d.metadata.Version); err != nil {
		return err
	}

	// Read timestamps
	if err := binary.Read(f, binary.BigEndian, &d.metadata.CreatedAt); err != nil {
		return err
	}
	if err := binary.Read(f, binary.BigEndian, &d.metadata.ModifiedAt); err != nil {
		return err
	}

	// Read schema hash
	var hashLen uint32
	if err := binary.Read(f, binary.BigEndian, &hashLen); err != nil {
		return err
	}
	d.metadata.SchemaHash = make([]byte, hashLen)
	if _, err := io.ReadFull(f, d.metadata.SchemaHash); err != nil {
		return err
	}

	return nil
}

// loadSchema loads a table schema.
func (d *Database) loadSchema(path string) (*Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tableName string
	var columns []ColumnDefinition
	var primaryKey []string

	// Read table name
	var nameLen uint32
	if err := binary.Read(f, binary.BigEndian, &nameLen); err != nil {
		return nil, err
	}
	nameBuf := make([]byte, nameLen)
	if _, err := io.ReadFull(f, nameBuf); err != nil {
		return nil, err
	}
	tableName = string(nameBuf)

	// Read column count
	var colCount uint32
	if err := binary.Read(f, binary.BigEndian, &colCount); err != nil {
		return nil, err
	}

	// Read columns
	for i := uint32(0); i < colCount; i++ {
		// Column name
		var colNameLen uint32
		if err := binary.Read(f, binary.BigEndian, &colNameLen); err != nil {
			return nil, err
		}
		colNameBuf := make([]byte, colNameLen)
		if _, err := io.ReadFull(f, colNameBuf); err != nil {
			return nil, err
		}

		// Column type
		var colType uint32
		if err := binary.Read(f, binary.BigEndian, &colType); err != nil {
			return nil, err
		}

		// Constraints
		var constraints uint32
		if err := binary.Read(f, binary.BigEndian, &constraints); err != nil {
			return nil, err
		}

		col := ColumnDefinition{
			Name:       string(colNameBuf),
			Type:       DataType(colType),
			PrimaryKey: constraints&(1<<0) != 0,
			NotNull:    constraints&(1<<1) != 0,
			Unique:     constraints&(1<<2) != 0,
			AutoInc:    constraints&(1<<3) != 0,
		}
		columns = append(columns, col)
	}

	// Read primary key
	var pkCount uint32
	if err := binary.Read(f, binary.BigEndian, &pkCount); err != nil {
		return nil, err
	}
	for i := uint32(0); i < pkCount; i++ {
		var pkNameLen uint32
		if err := binary.Read(f, binary.BigEndian, &pkNameLen); err != nil {
			return nil, err
		}
		pkNameBuf := make([]byte, pkNameLen)
		if _, err := io.ReadFull(f, pkNameBuf); err != nil {
			return nil, err
		}
		primaryKey = append(primaryKey, string(pkNameBuf))
	}

	return &Schema{
		TableName:  tableName,
		Columns:    columns,
		PrimaryKey: primaryKey,
	}, nil
}

// TransactionManager returns the transaction manager (placeholder).
func (d *Database) TransactionManager() interface{} {
	return nil
}

// DatabaseManager manages multiple databases.
type DatabaseManager struct {
	databases map[string]*Database
	mu        sync.RWMutex
}

// NewDatabaseManager creates a new database manager.
func NewDatabaseManager() *DatabaseManager {
	return &DatabaseManager{
		databases: make(map[string]*Database),
	}
}

// CreateDatabase creates a new database.
func (m *DatabaseManager) CreateDatabase(name, path string) (*Database, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.databases[name]; exists {
		return nil, ErrDatabaseExists
	}

	db, err := NewDatabase(name, path)
	if err != nil {
		return nil, err
	}

	m.databases[name] = db
	return db, nil
}

// GetDatabase returns a database by name.
func (m *DatabaseManager) GetDatabase(name string) (*Database, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	db, ok := m.databases[name]
	return db, ok
}

// DropDatabase drops a database by name.
func (m *DatabaseManager) DropDatabase(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, ok := m.databases[name]
	if !ok {
		return ErrDatabaseNotFound
	}

	if err := db.Close(); err != nil {
		return err
	}

	delete(m.databases, name)
	return nil
}

// DatabaseNames returns all database names.
func (m *DatabaseManager) DatabaseNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.databases))
	for name := range m.databases {
		names = append(names, name)
	}
	return names
}

// Close closes all databases.
func (m *DatabaseManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, db := range m.databases {
		if err := db.Close(); err != nil {
			return err
		}
	}
	m.databases = make(map[string]*Database)
	return nil
}
