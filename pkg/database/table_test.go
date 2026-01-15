package database

import (
	"fmt"
	"io"
	"testing"
)

func TestNewTable(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText, NotNull: true},
			{Name: "age", Type: DataTypeInteger},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	if table.Name() != "users" {
		t.Errorf("Table.Name() = %s, want %s", table.Name(), "users")
	}

	if table.RowCount() != 0 {
		t.Errorf("Table.RowCount() = %d, want %d", table.RowCount(), 0)
	}
}

func TestTableInsert(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText, NotNull: true},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Insert a row
	v1, _ := NewValue(int64(1))
	v2, _ := NewValue("Alice")
	id, err := table.Insert([]Value{v1, v2})
	if err != nil {
		t.Fatalf("Table.Insert() error = %v", err)
	}

	if id != 1 {
		t.Errorf("Table.Insert() returned id = %d, want %d", id, 1)
	}

	if table.RowCount() != 1 {
		t.Errorf("Table.RowCount() = %d, want %d", table.RowCount(), 1)
	}

	// Insert another row
	v3, _ := NewValue(int64(2))
	v4, _ := NewValue("Bob")
	id2, err := table.Insert([]Value{v3, v4})
	if err != nil {
		t.Fatalf("Table.Insert() error = %v", err)
	}
	if id2 != 2 {
		t.Errorf("Table.Insert() returned id = %d, want %d", id2, 2)
	}
}

func TestTableGet(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Insert a row
	v1, _ := NewValue(int64(1))
	v2, _ := NewValue("Alice")
	_, err = table.Insert([]Value{v1, v2})
	if err != nil {
		t.Fatalf("Table.Insert() error = %v", err)
	}

	// Get the row
	row, err := table.Get(1)
	if err != nil {
		t.Fatalf("Table.Get() error = %v", err)
	}

	if row.ID != 1 {
		t.Errorf("Row.ID = %d, want %d", row.ID, 1)
	}

	// Get non-existing row
	_, err = table.Get(999)
	if err != ErrRowNotFound {
		t.Errorf("Table.Get() non-existing error = %v, want %v", err, ErrRowNotFound)
	}
}

func TestTableUpdate(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Insert a row
	v1, _ := NewValue(int64(1))
	v2, _ := NewValue("Alice")
	_, err = table.Insert([]Value{v1, v2})
	if err != nil {
		t.Fatalf("Table.Insert() error = %v", err)
	}

	// Update the row
	v3, _ := NewValue(int64(1))
	v4, _ := NewValue("Alice Smith")
	err = table.Update(1, []Value{v3, v4})
	if err != nil {
		t.Fatalf("Table.Update() error = %v", err)
	}

	// Verify update
	row, err := table.Get(1)
	if err != nil {
		t.Fatalf("Table.Get() error = %v", err)
	}
	if row.Values[1].Str != "Alice Smith" {
		t.Errorf("Row name = %s, want %s", row.Values[1].Str, "Alice Smith")
	}

	// Update non-existing row
	v5, _ := NewValue(int64(999))
	v6, _ := NewValue("Nobody")
	err = table.Update(999, []Value{v5, v6})
	if err != ErrRowNotFound {
		t.Errorf("Table.Update() non-existing error = %v, want %v", err, ErrRowNotFound)
	}
}

func TestTableDelete(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Insert a row
	v1, _ := NewValue(int64(1))
	v2, _ := NewValue("Alice")
	_, err = table.Insert([]Value{v1, v2})
	if err != nil {
		t.Fatalf("Table.Insert() error = %v", err)
	}

	// Delete the row
	err = table.Delete(1)
	if err != nil {
		t.Fatalf("Table.Delete() error = %v", err)
	}

	if table.RowCount() != 0 {
		t.Errorf("Table.RowCount() = %d, want %d", table.RowCount(), 0)
	}

	// Verify deletion
	_, err = table.Get(1)
	if err != ErrRowNotFound {
		t.Errorf("Table.Get() after delete error = %v, want %v", err, ErrRowNotFound)
	}

	// Delete non-existing row
	err = table.Delete(999)
	if err != ErrRowNotFound {
		t.Errorf("Table.Delete() non-existing error = %v, want %v", err, ErrRowNotFound)
	}
}

func TestTableSelect(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "age", Type: DataTypeInteger},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Insert rows with different ages
	for i := 1; i <= 10; i++ {
		v1, _ := NewValue(int64(i))
		v2, _ := NewValue(int64(i * 10))
		_, err = table.Insert([]Value{v1, v2})
		if err != nil {
			t.Fatalf("Table.Insert() error = %v", err)
		}
	}

	// Select rows where age > 60
	rows, err := table.Select(func(row *Row) bool {
		return row.Values[1].Int > 60
	})
	if err != nil {
		t.Fatalf("Table.Select() error = %v", err)
	}

	if len(rows) != 4 {
		t.Errorf("Table.Select() returned %d rows, want %d", len(rows), 4)
	}

	// Select all rows
	allRows, err := table.Select(nil)
	if err != nil {
		t.Fatalf("Table.Select(nil) error = %v", err)
	}
	if len(allRows) != 10 {
		t.Errorf("Table.Select(nil) returned %d rows, want %d", len(allRows), 10)
	}
}

func TestTableTruncate(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Insert some rows
	for i := 1; i <= 5; i++ {
		v1, _ := NewValue(int64(i))
		v2, _ := NewValue("User")
		_, err = table.Insert([]Value{v1, v2})
		if err != nil {
			t.Fatalf("Table.Insert() error = %v", err)
		}
	}

	// Truncate
	err = table.Truncate()
	if err != nil {
		t.Fatalf("Table.Truncate() error = %v", err)
	}

	if table.RowCount() != 0 {
		t.Errorf("Table.RowCount() = %d, want %d", table.RowCount(), 0)
	}

	// Verify rows are gone
	_, err = table.Get(1)
	if err != ErrRowNotFound {
		t.Errorf("Table.Get() after truncate error = %v, want %v", err, ErrRowNotFound)
	}
}

func TestTableClose(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Close the table
	if err := table.Close(); err != nil {
		t.Fatalf("Table.Close() error = %v", err)
	}

	// Operations after close should fail
	v1, _ := NewValue(int64(1))
	_, err = table.Insert([]Value{v1})
	if err != ErrTableClosed {
		t.Errorf("Insert() after close error = %v, want %v", err, ErrTableClosed)
	}

	_, err = table.Get(1)
	if err != ErrTableClosed {
		t.Errorf("Get() after close error = %v, want %v", err, ErrTableClosed)
	}
}

func TestTableManager(t *testing.T) {
	mgr := NewTableManager()

	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText},
		},
		PrimaryKey: []string{"id"},
	}

	// Create table
	table, err := mgr.CreateTable("users", schema)
	if err != nil {
		t.Fatalf("CreateTable() error = %v", err)
	}

	// Get table
	gotTable, ok := mgr.GetTable("users")
	if !ok {
		t.Error("GetTable() returned false, want true")
	}
	if gotTable != table {
		t.Error("GetTable() returned different table")
	}

	// Table names
	names := mgr.TableNames()
	if len(names) != 1 || names[0] != "users" {
		t.Errorf("TableNames() = %v, want [users]", names)
	}

	// Duplicate table
	_, err = mgr.CreateTable("users", schema)
	if err == nil {
		t.Error("CreateTable() duplicate should fail")
	}

	// Drop table
	err = mgr.DropTable("users")
	if err != nil {
		t.Fatalf("DropTable() error = %v", err)
	}

	// Verify dropped
	_, ok = mgr.GetTable("users")
	if ok {
		t.Error("GetTable() after drop returned true, want false")
	}

	// Drop non-existing table
	err = mgr.DropTable("nonexistent")
	if err != ErrTableNotFound {
		t.Errorf("DropTable() non-existing error = %v, want %v", err, ErrTableNotFound)
	}

	// Close manager
	if err := mgr.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestTableCreateIndex(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "email", Type: DataTypeText},
			{Name: "name", Type: DataTypeText},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Insert some rows
	for i := 1; i <= 5; i++ {
		v1, _ := NewValue(int64(i))
		v2, _ := NewValue(fmt.Sprintf("user%d@example.com", i))
		v3, _ := NewValue(fmt.Sprintf("User %d", i))
		_, err = table.Insert([]Value{v1, v2, v3})
		if err != nil {
			t.Fatalf("Table.Insert() error = %v", err)
		}
	}

	// Create index on email
	err = table.CreateIndex("idx_email", []string{"email"}, true)
	if err != nil {
		t.Fatalf("CreateIndex() error = %v", err)
	}

	// Verify index exists
	idx, ok := table.GetIndex("idx_email")
	if !ok {
		t.Error("GetIndex() returned false, want true")
	}
	if idx.Name() != "idx_email" {
		t.Errorf("Index.Name() = %s, want %s", idx.Name(), "idx_email")
	}

	// Drop index
	err = table.DropIndex("idx_email")
	if err != nil {
		t.Fatalf("DropIndex() error = %v", err)
	}

	// Verify dropped
	_, ok = table.GetIndex("idx_email")
	if ok {
		t.Error("GetIndex() after drop returned true, want false")
	}
}

func TestRowSerializeDeserialize(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger},
			{Name: "name", Type: DataTypeText},
			{Name: "balance", Type: DataTypeFloat},
			{Name: "active", Type: DataTypeBoolean},
		},
	}

	v1, _ := NewValue(int64(123))
	v2, _ := NewValue("Test User")
	v3, _ := NewValue(99.99)
	v4, _ := NewValue(true)

	original := &Row{
		ID:       42,
		SchemaID: 1,
		Values: []Value{
			v1, v2, v3, v4,
		},
	}

	// Serialize
	data, err := original.Serialize(schema)
	if err != nil {
		t.Fatalf("Row.Serialize() error = %v", err)
	}

	// Deserialize
	restored, err := DeserializeRow(data, schema)
	if err != nil {
		t.Fatalf("DeserializeRow() error = %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("Row.ID = %d, want %d", restored.ID, original.ID)
	}
	if restored.SchemaID != original.SchemaID {
		t.Errorf("Row.SchemaID = %d, want %d", restored.SchemaID, original.SchemaID)
	}
	if len(restored.Values) != len(original.Values) {
		t.Errorf("Row.Values length = %d, want %d", len(restored.Values), len(original.Values))
	}
}

func TestTableIterator(t *testing.T) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}

	// Insert 10 rows
	for i := 1; i <= 10; i++ {
		v1, _ := NewValue(int64(i))
		_, err = table.Insert([]Value{v1})
		if err != nil {
			t.Fatalf("Table.Insert() error = %v", err)
		}
	}

	// Iterate
	iter := NewTableIterator(table)
	count := 0
	for {
		row, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Iterator.Next() error = %v", err)
		}
		if row == nil {
			break
		}
		count++
	}

	if count != 10 {
		t.Errorf("Iterator returned %d rows, want %d", count, 10)
	}
}

func BenchmarkTableInsert(b *testing.B) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText},
			{Name: "age", Type: DataTypeInteger},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		b.Fatalf("NewTable() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v1, _ := NewValue(int64(i))
		v2, _ := NewValue(fmt.Sprintf("User %d", i))
		v3, _ := NewValue(int64(i % 100))
		table.Insert([]Value{v1, v2, v3})
	}
}

func BenchmarkTableGet(b *testing.B) {
	schema := &Schema{
		TableName: "test_users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: DataTypeText},
		},
		PrimaryKey: []string{"id"},
	}

	table, err := NewTable("users", schema)
	if err != nil {
		b.Fatalf("NewTable() error = %v", err)
	}

	// Insert some data
	for i := 0; i < 1000; i++ {
		v1, _ := NewValue(int64(i))
		v2, _ := NewValue(fmt.Sprintf("User %d", i))
		table.Insert([]Value{v1, v2})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table.Get(RowID(i % 1000))
	}
}
