package database

import (
	"testing"
)

func TestDataTypeString(t *testing.T) {
	tests := []struct {
		dt   DataType
		want string
	}{
		{DataTypeNull, "NULL"},
		{DataTypeInteger, "INTEGER"},
		{DataTypeFloat, "FLOAT"},
		{DataTypeBoolean, "BOOLEAN"},
		{DataTypeText, "TEXT"},
		{DataTypeBlob, "BLOB"},
		{DataTypeDate, "DATE"},
		{DataTypeDateTime, "DATETIME"},
		{DataType(100), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.dt.String(); got != tt.want {
			t.Errorf("DataType.String() = %v, want %v", got, tt.want)
		}
	}
}

func TestParseDataType(t *testing.T) {
	tests := []struct {
		input string
		want  DataType
		err   bool
	}{
		{"NULL", DataTypeNull, false},
		{"INTEGER", DataTypeInteger, false},
		{"INT", DataTypeInteger, false},
		{"BIGINT", DataTypeInteger, false},
		{"FLOAT", DataTypeFloat, false},
		{"DOUBLE", DataTypeFloat, false},
		{"BOOLEAN", DataTypeBoolean, false},
		{"BOOL", DataTypeBoolean, false},
		{"TEXT", DataTypeText, false},
		{"VARCHAR", DataTypeText, false},
		{"BLOB", DataTypeBlob, false},
		{"DATE", DataTypeDate, false},
		{"DATETIME", DataTypeDateTime, false},
		{"UNKNOWN", DataTypeNull, true},
	}

	for _, tt := range tests {
		got, err := ParseDataType(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("ParseDataType(%s) error = %v, wantErr %v", tt.input, err, tt.err)
			continue
		}
		if !tt.err && got != tt.want {
			t.Errorf("ParseDataType(%s) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestColumnDefinitionValidate(t *testing.T) {
	tests := []struct {
		name    string
		col     ColumnDefinition
		wantErr bool
	}{
		{
			name: "valid column",
			col: ColumnDefinition{
				Name: "id",
				Type: DataTypeInteger,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			col: ColumnDefinition{
				Name: "",
				Type: DataTypeInteger,
			},
			wantErr: true,
		},
		{
			name: "autoincrement requires integer",
			col: ColumnDefinition{
				Name:    "id",
				Type:    DataTypeText,
				AutoInc: true,
			},
			wantErr: true,
		},
		{
			name: "autoincrement requires primary key",
			col: ColumnDefinition{
				Name:    "id",
				Type:    DataTypeInteger,
				AutoInc: true,
			},
			wantErr: true,
		},
		{
			name: "valid primary key with autoincrement",
			col: ColumnDefinition{
				Name:       "id",
				Type:       DataTypeInteger,
				AutoInc:    true,
				PrimaryKey: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.col.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ColumnDefinition.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewValue(t *testing.T) {
	tests := []struct {
		input interface{}
		want  Value
	}{
		{nil, Value{Type: DataTypeNull}},
		{int(42), Value{Type: DataTypeInteger, Int: 42}},
		{int64(123), Value{Type: DataTypeInteger, Int: 123}},
		{float64(3.14), Value{Type: DataTypeFloat, Float: 3.14}},
		{true, Value{Type: DataTypeBoolean, Bool: true}},
		{false, Value{Type: DataTypeBoolean, Bool: false}},
		{"hello", Value{Type: DataTypeText, Str: "hello"}},
		{[]byte{1, 2, 3}, Value{Type: DataTypeBlob, Blob: []byte{1, 2, 3}}},
	}

	for _, tt := range tests {
		got, err := NewValue(tt.input)
		if err != nil {
			t.Errorf("NewValue(%v) error = %v", tt.input, err)
			continue
		}
		if got.Type != tt.want.Type {
			t.Errorf("NewValue(%v).Type = %v, want %v", tt.input, got.Type, tt.want.Type)
		}
	}
}

func TestValueIsNull(t *testing.T) {
	tests := []struct {
		value Value
		want  bool
	}{
		{Value{Type: DataTypeNull}, true},
		{Value{Type: DataTypeInteger, Int: 42}, false},
		{Value{Type: DataTypeText, Str: "test"}, false},
	}

	for _, tt := range tests {
		if got := tt.value.IsNull(); got != tt.want {
			t.Errorf("Value.IsNull() = %v, want %v", got, tt.want)
		}
	}
}

func TestValueCompare(t *testing.T) {
	tests := []struct {
		a    Value
		b    Value
		want int
		err  bool
	}{
		{
			Value{Type: DataTypeNull},
			Value{Type: DataTypeNull},
			0, false,
		},
		{
			Value{Type: DataTypeInteger, Int: 1},
			Value{Type: DataTypeInteger, Int: 2},
			-1, false,
		},
		{
			Value{Type: DataTypeInteger, Int: 5},
			Value{Type: DataTypeInteger, Int: 5},
			0, false,
		},
		{
			Value{Type: DataTypeInteger, Int: 10},
			Value{Type: DataTypeInteger, Int: 5},
			1, false,
		},
		{
			Value{Type: DataTypeFloat, Float: 1.5},
			Value{Type: DataTypeFloat, Float: 2.5},
			-1, false,
		},
		{
			Value{Type: DataTypeText, Str: "a"},
			Value{Type: DataTypeText, Str: "b"},
			-1, false,
		},
		{
			Value{Type: DataTypeInteger},
			Value{Type: DataTypeText},
			0, true,
		},
	}

	for _, tt := range tests {
		got, err := tt.a.Compare(tt.b)
		if (err != nil) != tt.err {
			t.Errorf("Value.Compare() error = %v, wantErr %v", err, tt.err)
			continue
		}
		if !tt.err && got != tt.want {
			t.Errorf("Value.Compare() = %v, want %v", got, tt.want)
		}
	}
}

func TestValueSerializeDeserialize(t *testing.T) {
	values := []Value{
		{Type: DataTypeNull},
		{Type: DataTypeInteger, Int: 42},
		{Type: DataTypeInteger, Int: -100},
		{Type: DataTypeFloat, Float: 3.14159},
		{Type: DataTypeBoolean, Bool: true},
		{Type: DataTypeBoolean, Bool: false},
		{Type: DataTypeText, Str: "Hello, World!"},
		{Type: DataTypeBlob, Blob: []byte{1, 2, 3, 4, 5}},
	}

	for _, original := range values {
		serialized, err := original.Serialize()
		if err != nil {
			t.Errorf("Value.Serialize() error = %v", err)
			continue
		}

		deserialized, err := Deserialize(serialized)
		if err != nil {
			t.Errorf("Deserialize() error = %v", err)
			continue
		}

		if original.Type != deserialized.Type {
			t.Errorf("Type mismatch: got %v, want %v", deserialized.Type, original.Type)
		}

		switch original.Type {
		case DataTypeInteger:
			if deserialized.Int != original.Int {
				t.Errorf("Integer mismatch: got %d, want %d", deserialized.Int, original.Int)
			}
		case DataTypeFloat:
			if deserialized.Float != original.Float {
				t.Errorf("Float mismatch: got %f, want %f", deserialized.Float, original.Float)
			}
		case DataTypeBoolean:
			if deserialized.Bool != original.Bool {
				t.Errorf("Boolean mismatch: got %v, want %v", deserialized.Bool, original.Bool)
			}
		case DataTypeText:
			if deserialized.Str != original.Str {
				t.Errorf("Text mismatch: got %s, want %s", deserialized.Str, original.Str)
			}
		case DataTypeBlob:
			if string(deserialized.Blob) != string(original.Blob) {
				t.Errorf("Blob mismatch: got %v, want %v", deserialized.Blob, original.Blob)
			}
		}
	}
}

func TestSchemaValidate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Schema
		wantErr bool
	}{
		{
			name: "valid schema",
			schema: Schema{
				TableName: "users",
				Columns: []ColumnDefinition{
					{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
					{Name: "name", Type: DataTypeText, NotNull: true},
					{Name: "age", Type: DataTypeInteger},
				},
				PrimaryKey: []string{"id"},
			},
			wantErr: false,
		},
		{
			name: "empty table name",
			schema: Schema{
				TableName: "",
				Columns:   []ColumnDefinition{{Name: "id", Type: DataTypeInteger}},
			},
			wantErr: true,
		},
		{
			name: "no columns",
			schema: Schema{
				TableName: "empty_table",
				Columns:   []ColumnDefinition{},
			},
			wantErr: true,
		},
		{
			name: "duplicate column",
			schema: Schema{
				TableName: "test",
				Columns: []ColumnDefinition{
					{Name: "id", Type: DataTypeInteger},
					{Name: "id", Type: DataTypeText},
				},
			},
			wantErr: true,
		},
		{
			name: "primary key column not found",
			schema: Schema{
				TableName:  "test",
				Columns:    []ColumnDefinition{{Name: "id", Type: DataTypeInteger}},
				PrimaryKey: []string{"missing"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Schema.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaHelpers(t *testing.T) {
	schema := Schema{
		TableName: "users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger},
			{Name: "name", Type: DataTypeText},
			{Name: "email", Type: DataTypeText},
		},
	}

	// Test ColumnCount
	if got := schema.ColumnCount(); got != 3 {
		t.Errorf("Schema.ColumnCount() = %v, want %v", got, 3)
	}

	// Test GetColumn
	col, found := schema.GetColumn("name")
	if !found {
		t.Error("Schema.GetColumn(\"name\") = false, want true")
	}
	if col.Name != "name" {
		t.Errorf("Schema.GetColumn(\"name\").Name = %v, want %v", col.Name, "name")
	}

	// Test GetColumnIndex
	if got := schema.GetColumnIndex("email"); got != 2 {
		t.Errorf("Schema.GetColumnIndex(\"email\") = %v, want %v", got, 2)
	}

	// Test HasColumn
	if !schema.HasColumn("id") {
		t.Error("Schema.HasColumn(\"id\") = false, want true")
	}
	if schema.HasColumn("missing") {
		t.Error("Schema.HasColumn(\"missing\") = true, want false")
	}
}
