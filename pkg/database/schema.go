// Package database provides a SQL database engine with ACID transactions.
package database

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unicode/utf8"
)

// DataType represents the type of a column in a table.
type DataType int

const (
	// DataTypeNull represents a NULL value.
	DataTypeNull DataType = iota
	// DataTypeInteger represents a 64-bit integer.
	DataTypeInteger
	// DataTypeFloat represents a 64-bit floating point number.
	DataTypeFloat
	// DataTypeBoolean represents a boolean value.
	DataTypeBoolean
	// DataTypeText represents a variable-length text string.
	DataTypeText
	// DataTypeBlob represents binary data.
	DataTypeBlob
	// DataTypeDate represents a date value.
	DataTypeDate
	// DataTypeDateTime represents a date-time value.
	DataTypeDateTime
)

var (
	// ErrInvalidDataType indicates an unknown or unsupported data type.
	ErrInvalidDataType = errors.New("invalid data type")
	// ErrNullValue indicates a NULL value where a non-NULL is required.
	ErrNullValue = errors.New("null value not allowed")
	// ErrValueTooLarge indicates a value exceeds the maximum size.
	ErrValueTooLarge = errors.New("value too large")
	// ErrTypeMismatch indicates a value cannot be converted to the column type.
	ErrTypeMismatch = errors.New("type mismatch")
)

// String returns the string representation of the data type.
func (dt DataType) String() string {
	switch dt {
	case DataTypeNull:
		return "NULL"
	case DataTypeInteger:
		return "INTEGER"
	case DataTypeFloat:
		return "FLOAT"
	case DataTypeBoolean:
		return "BOOLEAN"
	case DataTypeText:
		return "TEXT"
	case DataTypeBlob:
		return "BLOB"
	case DataTypeDate:
		return "DATE"
	case DataTypeDateTime:
		return "DATETIME"
	default:
		return "UNKNOWN"
	}
}

// ParseDataType parses a string into a DataType.
func ParseDataType(s string) (DataType, error) {
	switch s {
	case "NULL":
		return DataTypeNull, nil
	case "INTEGER", "INT", "BIGINT", "SMALLINT":
		return DataTypeInteger, nil
	case "FLOAT", "DOUBLE", "REAL":
		return DataTypeFloat, nil
	case "BOOLEAN", "BOOL":
		return DataTypeBoolean, nil
	case "TEXT", "VARCHAR", "CHAR", "STRING":
		return DataTypeText, nil
	case "BLOB":
		return DataTypeBlob, nil
	case "DATE":
		return DataTypeDate, nil
	case "DATETIME", "TIMESTAMP":
		return DataTypeDateTime, nil
	default:
		return DataTypeNull, ErrInvalidDataType
	}
}

// ColumnConstraint represents a constraint on a column.
type ColumnConstraint int

const (
	// ConstraintNone represents no constraint.
	ConstraintNone ColumnConstraint = iota
	// ConstraintNotNull indicates the column cannot be NULL.
	ConstraintNotNull
	// ConstraintPrimaryKey indicates the column is part of the primary key.
	ConstraintPrimaryKey
	// ConstraintUnique indicates the column must have unique values.
	ConstraintUnique
	// ConstraintAutoIncrement indicates the column auto-increments.
	ConstraintAutoIncrement
	// ConstraintDefault indicates the column has a default value.
	ConstraintDefault
)

// String returns the string representation of the constraint.
func (cc ColumnConstraint) String() string {
	switch cc {
	case ConstraintNone:
		return ""
	case ConstraintNotNull:
		return "NOT NULL"
	case ConstraintPrimaryKey:
		return "PRIMARY KEY"
	case ConstraintUnique:
		return "UNIQUE"
	case ConstraintAutoIncrement:
		return "AUTOINCREMENT"
	case ConstraintDefault:
		return "DEFAULT"
	default:
		return ""
	}
}

// ColumnDefinition represents a column in a table schema.
type ColumnDefinition struct {
	Name       string           // Column name
	Type       DataType         // Column data type
	Constraint ColumnConstraint // Column constraint
	Default    Value            // Default value (if ConstraintDefault)
	PrimaryKey bool             // Whether this column is a primary key
	NotNull    bool             // Whether NULL values are allowed
	Unique     bool             // Whether values must be unique
	AutoInc    bool             // Whether this is an auto-increment column
}

// Validate validates the column definition.
func (c ColumnDefinition) Validate() error {
	if c.Name == "" {
		return errors.New("column name cannot be empty")
	}
	if !utf8.ValidString(c.Name) {
		return errors.New("column name must be valid UTF-8")
	}
	if c.Type == DataTypeNull && c.Constraint != ConstraintNone {
		return errors.New("NULL type cannot have constraints")
	}
	if c.AutoInc && c.Type != DataTypeInteger {
		return errors.New("AUTOINCREMENT requires INTEGER type")
	}
	if c.AutoInc && !c.PrimaryKey {
		return errors.New("AUTOINCREMENT requires PRIMARY KEY")
	}
	return nil
}

// Column returns a copy of the column definition.
func (c ColumnDefinition) Column() ColumnDefinition {
	return c
}

// Value represents a value in the database.
type Value struct {
	Type  DataType // Type of the value
	Int   int64    // Integer value
	Float float64  // Float value
	Bool  bool     // Boolean value
	Str   string   // String value
	Blob  []byte   // Binary data
}

// NewValue creates a new Value from a Go value.
func NewValue(v interface{}) (Value, error) {
	val := Value{}
	switch v := v.(type) {
	case nil:
		val.Type = DataTypeNull
	case int:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case int8:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case int16:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case int32:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case int64:
		val.Type = DataTypeInteger
		val.Int = v
	case uint:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case uint8:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case uint16:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case uint32:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case uint64:
		val.Type = DataTypeInteger
		val.Int = int64(v)
	case float32:
		val.Type = DataTypeFloat
		val.Float = float64(v)
	case float64:
		val.Type = DataTypeFloat
		val.Float = v
	case bool:
		val.Type = DataTypeBoolean
		val.Bool = v
	case string:
		val.Type = DataTypeText
		val.Str = v
	case []byte:
		val.Type = DataTypeBlob
		val.Blob = v
	default:
		return Value{}, fmt.Errorf("unsupported type: %T", v)
	}
	return val, nil
}

// IsNull returns true if the value is NULL.
func (v Value) IsNull() bool {
	return v.Type == DataTypeNull
}

// Compare compares two values and returns -1, 0, or 1.
func (v Value) Compare(other Value) (int, error) {
	if v.Type != other.Type {
		return 0, ErrTypeMismatch
	}
	switch v.Type {
	case DataTypeNull:
		return 0, nil
	case DataTypeInteger:
		if v.Int < other.Int {
			return -1, nil
		}
		if v.Int > other.Int {
			return 1, nil
		}
		return 0, nil
	case DataTypeFloat:
		if v.Float < other.Float {
			return -1, nil
		}
		if v.Float > other.Float {
			return 1, nil
		}
		return 0, nil
	case DataTypeBoolean:
		if v.Bool == other.Bool {
			return 0, nil
		}
		return -1, nil
	case DataTypeText:
		if v.Str < other.Str {
			return -1, nil
		}
		if v.Str > other.Str {
			return 1, nil
		}
		return 0, nil
	case DataTypeBlob:
		return bytesCompare(v.Blob, other.Blob), nil
	default:
		return 0, ErrInvalidDataType
	}
}

// bytesCompare compares two byte slices.
func bytesCompare(a, b []byte) int {
	n := len(a)
	if n > len(b) {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

// Serialize serializes the value to binary format.
func (v Value) Serialize() ([]byte, error) {
	buf := make([]byte, 0, 16)
	buf = append(buf, byte(v.Type))
	switch v.Type {
	case DataTypeNull:
		return buf, nil
	case DataTypeInteger:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], uint64(v.Int))
		return append(buf, b[:]...), nil
	case DataTypeFloat:
		var b [8]byte
		bits := math.Float64bits(v.Float)
		binary.BigEndian.PutUint64(b[:], bits)
		return append(buf, b[:]...), nil
	case DataTypeBoolean:
		if v.Bool {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
		return buf, nil
	case DataTypeText:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(len(v.Str)))
		buf = append(buf, b[:]...)
		return append(buf, v.Str...), nil
	case DataTypeBlob:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(len(v.Blob)))
		buf = append(buf, b[:]...)
		return append(buf, v.Blob...), nil
	default:
		return nil, ErrInvalidDataType
	}
}

// Deserialize deserializes a value from binary format.
func Deserialize(data []byte) (Value, error) {
	if len(data) == 0 {
		return Value{}, errors.New("invalid data")
	}
	v := Value{}
	v.Type = DataType(data[0])
	data = data[1:]
	switch v.Type {
	case DataTypeNull:
		return v, nil
	case DataTypeInteger:
		if len(data) < 8 {
			return Value{}, errors.New("insufficient data for INTEGER")
		}
		v.Int = int64(binary.BigEndian.Uint64(data[:8]))
		return v, nil
	case DataTypeFloat:
		if len(data) < 8 {
			return Value{}, errors.New("insufficient data for FLOAT")
		}
		bits := binary.BigEndian.Uint64(data[:8])
		v.Float = math.Float64frombits(bits)
		return v, nil
	case DataTypeBoolean:
		if len(data) < 1 {
			return Value{}, errors.New("insufficient data for BOOLEAN")
		}
		v.Bool = data[0] != 0
		return v, nil
	case DataTypeText:
		if len(data) < 4 {
			return Value{}, errors.New("insufficient data for TEXT length")
		}
		n := int(binary.BigEndian.Uint32(data[:4]))
		if len(data) < 4+n {
			return Value{}, errors.New("insufficient data for TEXT")
		}
		v.Str = string(data[4 : 4+n])
		return v, nil
	case DataTypeBlob:
		if len(data) < 4 {
			return Value{}, errors.New("insufficient data for BLOB length")
		}
		n := int(binary.BigEndian.Uint32(data[:4]))
		if len(data) < 4+n {
			return Value{}, errors.New("insufficient data for BLOB")
		}
		v.Blob = make([]byte, n)
		copy(v.Blob, data[4:4+n])
		return v, nil
	default:
		return Value{}, ErrInvalidDataType
	}
}

// Schema represents the schema of a table.
type Schema struct {
	TableName   string             // Name of the table
	Columns     []ColumnDefinition // Column definitions
	PrimaryKey  []string           // Names of primary key columns
	Indexes     []IndexDefinition  // Index definitions
	ForeignKeys []ForeignKey       // Foreign key constraints
	Constraints []TableConstraint  // Table-level constraints
}

// ColumnCount returns the number of columns.
func (s Schema) ColumnCount() int {
	return len(s.Columns)
}

// GetColumn returns the column definition by name.
func (s Schema) GetColumn(name string) (ColumnDefinition, bool) {
	for _, col := range s.Columns {
		if col.Name == name {
			return col, true
		}
	}
	return ColumnDefinition{}, false
}

// GetColumnIndex returns the index of a column by name.
func (s Schema) GetColumnIndex(name string) int {
	for i, col := range s.Columns {
		if col.Name == name {
			return i
		}
	}
	return -1
}

// HasColumn returns true if the schema has a column with the given name.
func (s Schema) HasColumn(name string) bool {
	return s.GetColumnIndex(name) >= 0
}

// Validate validates the schema.
func (s Schema) Validate() error {
	if s.TableName == "" {
		return errors.New("table name cannot be empty")
	}
	if !utf8.ValidString(s.TableName) {
		return errors.New("table name must be valid UTF-8")
	}
	if len(s.Columns) == 0 {
		return errors.New("table must have at least one column")
	}
	// Check for duplicate column names
	colNames := make(map[string]bool)
	for _, col := range s.Columns {
		if err := col.Validate(); err != nil {
			return fmt.Errorf("column %s: %w", col.Name, err)
		}
		if colNames[col.Name] {
			return fmt.Errorf("duplicate column name: %s", col.Name)
		}
		colNames[col.Name] = true
	}
	// Validate primary key columns exist
	for _, pkCol := range s.PrimaryKey {
		if !s.HasColumn(pkCol) {
			return fmt.Errorf("primary key column %s not found", pkCol)
		}
	}
	// Validate index columns exist
	for _, idx := range s.Indexes {
		for _, col := range idx.Columns {
			if !s.HasColumn(col) {
				return fmt.Errorf("index column %s not found", col)
			}
		}
	}
	return nil
}

// IndexDefinition represents an index on a table.
type IndexDefinition struct {
	Name    string   // Index name
	Table   string   // Table name
	Columns []string // Column names in the index
	Unique  bool     // Whether the index is unique
}

// ForeignKey represents a foreign key constraint.
type ForeignKey struct {
	Name              string   // Constraint name
	Columns           []string // Local column names
	RefTable          string   // Referenced table name
	RefColumns        []string // Referenced column names
	OnDelete          string   // ON DELETE action
	OnUpdate          string   // ON UPDATE action
	Deferrable        bool     // Whether constraint is deferrable
	InitiallyDeferred bool     // Whether initially deferred
}

// TableConstraint represents a table-level constraint.
type TableConstraint int

const (
	// TableConstraintNone represents no constraint.
	TableConstraintNone TableConstraint = iota
	// TableConstraintPrimaryKey represents a primary key constraint.
	TableConstraintPrimaryKey
	// TableConstraintUnique represents a unique constraint.
	TableConstraintUnique
	// TableConstraintCheck represents a CHECK constraint.
	TableConstraintCheck
	// TableConstraintForeignKey represents a foreign key constraint.
	TableConstraintForeignKey
)
