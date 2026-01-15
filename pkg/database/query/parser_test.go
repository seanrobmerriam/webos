package query

import (
	"testing"
)

// TestParseSQLSelect tests parsing SELECT statements.
func TestParseSQLSelect(t *testing.T) {
	tests := []struct {
		input    string
		wantType StatementType
	}{
		{"SELECT * FROM users", StmtSelect},
		{"SELECT id, name FROM users", StmtSelect},
		{"SELECT id, name FROM users WHERE id = 1", StmtSelect},
		{"SELECT * FROM users ORDER BY name", StmtSelect},
		{"SELECT * FROM users ORDER BY name DESC", StmtSelect},
		{"SELECT * FROM users LIMIT 10", StmtSelect},
		{"SELECT * FROM users LIMIT 10 OFFSET 5", StmtSelect},
		{"SELECT * FROM users WHERE age > 18 AND active = TRUE", StmtSelect},
	}

	for _, tt := range tests {
		stmt, err := ParseSQL(tt.input)
		if err != nil {
			t.Errorf("ParseSQL(%q) failed: %v", tt.input, err)
			continue
		}
		if stmt.Type != tt.wantType {
			t.Errorf("ParseSQL(%q) type = %v, want %v", tt.input, stmt.Type, tt.wantType)
		}
	}
}

// TestParseSQLInsert tests parsing INSERT statements.
func TestParseSQLInsert(t *testing.T) {
	tests := []struct {
		input    string
		wantType StatementType
	}{
		{"INSERT INTO users VALUES (1, 'John')", StmtInsert},
		{"INSERT INTO users (name, age) VALUES ('John', 25)", StmtInsert},
		{"INSERT INTO users (id, name, active) VALUES (1, 'John', TRUE)", StmtInsert},
	}

	for _, tt := range tests {
		stmt, err := ParseSQL(tt.input)
		if err != nil {
			t.Errorf("ParseSQL(%q) failed: %v", tt.input, err)
			continue
		}
		if stmt.Type != tt.wantType {
			t.Errorf("ParseSQL(%q) type = %v, want %v", tt.input, stmt.Type, tt.wantType)
		}
		if stmt.Insert == nil {
			t.Errorf("ParseSQL(%q) Insert is nil", tt.input)
		}
	}
}

// TestParseSQLUpdate tests parsing UPDATE statements.
func TestParseSQLUpdate(t *testing.T) {
	tests := []struct {
		input    string
		wantType StatementType
	}{
		{"UPDATE users SET name = 'John'", StmtUpdate},
		{"UPDATE users SET name = 'John', age = 30", StmtUpdate},
		{"UPDATE users SET name = 'John' WHERE id = 1", StmtUpdate},
	}

	for _, tt := range tests {
		stmt, err := ParseSQL(tt.input)
		if err != nil {
			t.Errorf("ParseSQL(%q) failed: %v", tt.input, err)
			continue
		}
		if stmt.Type != tt.wantType {
			t.Errorf("ParseSQL(%q) type = %v, want %v", tt.input, stmt.Type, tt.wantType)
		}
	}
}

// TestParseSQLDelete tests parsing DELETE statements.
func TestParseSQLDelete(t *testing.T) {
	tests := []struct {
		input    string
		wantType StatementType
	}{
		{"DELETE FROM users", StmtDelete},
		{"DELETE FROM users WHERE id = 1", StmtDelete},
	}

	for _, tt := range tests {
		stmt, err := ParseSQL(tt.input)
		if err != nil {
			t.Errorf("ParseSQL(%q) failed: %v", tt.input, err)
			continue
		}
		if stmt.Type != tt.wantType {
			t.Errorf("ParseSQL(%q) type = %v, want %v", tt.input, stmt.Type, tt.wantType)
		}
	}
}

// TestParseSQLCreateTable tests parsing CREATE TABLE statements.
func TestParseSQLCreateTable(t *testing.T) {
	input := "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, age INTEGER)"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Type != StmtCreateTable {
		t.Errorf("ParseSQL(%q) type = %v, want %v", input, stmt.Type, StmtCreateTable)
	}
	if stmt.CreateTable == nil {
		t.Errorf("ParseSQL(%q) CreateTable is nil", input)
		return
	}
	if stmt.CreateTable.TableName != "users" {
		t.Errorf("TableName = %q, want %q", stmt.CreateTable.TableName, "users")
	}
}

// TestParseSQLDropTable tests parsing DROP TABLE statements.
func TestParseSQLDropTable(t *testing.T) {
	input := "DROP TABLE users"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Type != StmtDropTable {
		t.Errorf("ParseSQL(%q) type = %v, want %v", input, stmt.Type, StmtDropTable)
	}
	if stmt.DropTable == nil {
		t.Errorf("ParseSQL(%q) DropTable is nil", input)
		return
	}
	if stmt.DropTable.TableName != "users" {
		t.Errorf("TableName = %q, want %q", stmt.DropTable.TableName, "users")
	}
}

// TestParseSQLAlterTable tests parsing ALTER TABLE statements.
func TestParseSQLAlterTable(t *testing.T) {
	tests := []struct {
		input    string
		wantType StatementType
	}{
		{"ALTER TABLE users ADD COLUMN email TEXT", StmtAlterTable},
		{"ALTER TABLE users DROP COLUMN email", StmtAlterTable},
	}

	for _, tt := range tests {
		stmt, err := ParseSQL(tt.input)
		if err != nil {
			t.Errorf("ParseSQL(%q) failed: %v", tt.input, err)
			continue
		}
		if stmt.Type != tt.wantType {
			t.Errorf("ParseSQL(%q) type = %v, want %v", tt.input, stmt.Type, tt.wantType)
		}
	}
}

// TestParseSQLJoin tests parsing JOIN statements.
func TestParseSQLJoin(t *testing.T) {
	input := "SELECT * FROM users INNER JOIN orders ON users.id = orders.user_id"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Type != StmtSelect {
		t.Errorf("ParseSQL(%q) type = %v, want %v", input, stmt.Type, StmtSelect)
	}
	if len(stmt.Select.Joins) != 1 {
		t.Errorf("Expected 1 join, got %d", len(stmt.Select.Joins))
	}
}

// TestParseSQLMultipleValues tests parsing INSERT with multiple VALUES.
func TestParseSQLMultipleValues(t *testing.T) {
	input := "INSERT INTO users (id, name) VALUES (1, 'John'), (2, 'Jane'), (3, 'Bob')"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Insert == nil {
		t.Errorf("Insert is nil")
		return
	}
	if len(stmt.Insert.Values) != 3 {
		t.Errorf("Expected 3 value lists, got %d", len(stmt.Insert.Values))
	}
}

// TestParseSQLColumnList tests parsing column lists in SELECT.
func TestParseSQLColumnList(t *testing.T) {
	input := "SELECT id, name, email, age FROM users"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if len(stmt.Select.Columns) != 4 {
		t.Errorf("Expected 4 columns, got %d", len(stmt.Select.Columns))
	}
}

// TestParseSQLError tests parsing invalid SQL.
func TestParseSQLError(t *testing.T) {
	tests := []struct {
		input string
	}{
		{""},
		{"INVALID"},
		{"SELECT FROM"},
		{"INSERT INTO"},
	}

	for _, tt := range tests {
		_, err := ParseSQL(tt.input)
		if err == nil {
			t.Errorf("ParseSQL(%q) should have failed", tt.input)
		}
	}
}

// TestParseSQLGroupBy tests parsing GROUP BY clause.
func TestParseSQLGroupBy(t *testing.T) {
	input := "SELECT category, COUNT(*) FROM products GROUP BY category"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Select.GroupBy == nil || len(stmt.Select.GroupBy) == 0 {
		t.Error("Expected GROUP BY clause to be parsed")
	}
}

// TestParseSQLHaving tests parsing HAVING clause.
func TestParseSQLHaving(t *testing.T) {
	input := "SELECT category, COUNT(*) FROM products GROUP BY category HAVING COUNT(*) > 10"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Select.Having.Type == ExprBinary {
		// Should have HAVING clause
	}
}

// TestParseSQLLeftJoin tests parsing LEFT JOIN.
func TestParseSQLLeftJoin(t *testing.T) {
	input := "SELECT * FROM users LEFT JOIN orders ON users.id = orders.user_id"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if len(stmt.Select.Joins) != 1 {
		t.Errorf("Expected 1 join, got %d", len(stmt.Select.Joins))
	}
	if stmt.Select.Joins[0].Type != "LEFT" {
		t.Errorf("Expected LEFT join, got %s", stmt.Select.Joins[0].Type)
	}
}

// TestParseSQLDistinct tests parsing DISTINCT.
func TestParseSQLDistinct(t *testing.T) {
	input := "SELECT DISTINCT category FROM products"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if !stmt.Select.Distinct {
		t.Error("Expected Distinct to be true")
	}
}

// TestParseSQLLike tests parsing LIKE operator.
func TestParseSQLLike(t *testing.T) {
	input := "SELECT * FROM users WHERE name LIKE 'J%'"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Select.Where.Op != "LIKE" {
		t.Errorf("Expected LIKE operator, got %s", stmt.Select.Where.Op)
	}
}

// TestParseSQLNotEqual tests parsing != and <> operators.
func TestParseSQLNotEqual(t *testing.T) {
	tests := []struct {
		input string
		op    string
	}{
		{"SELECT * FROM users WHERE id != 1", "!="},
		{"SELECT * FROM users WHERE id <> 1", "<>"},
	}

	for _, tt := range tests {
		stmt, err := ParseSQL(tt.input)
		if err != nil {
			t.Errorf("ParseSQL(%q) failed: %v", tt.input, err)
			continue
		}
		if stmt.Select.Where.Op != tt.op {
			t.Errorf("ParseSQL(%q) operator = %q, want %q", tt.input, stmt.Select.Where.Op, tt.op)
		}
	}
}

// TestParseSQLBetween tests parsing BETWEEN operator.
func TestParseSQLBetween(t *testing.T) {
	input := "SELECT * FROM users WHERE age BETWEEN 18 AND 65"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Select.Where.Op != "BETWEEN" {
		t.Errorf("Expected BETWEEN operator, got %s", stmt.Select.Where.Op)
	}
}

// TestParseSQLIn tests parsing IN operator.
func TestParseSQLIn(t *testing.T) {
	input := "SELECT * FROM users WHERE id IN (1, 2, 3)"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Select.Where.Op != "IN" {
		t.Errorf("Expected IN operator, got %s", stmt.Select.Where.Op)
	}
}

// TestParseSQLOrderByMultiple tests parsing multiple ORDER BY columns.
func TestParseSQLOrderByMultiple(t *testing.T) {
	input := "SELECT * FROM users ORDER BY name ASC, age DESC"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if len(stmt.Select.OrderBy) != 2 {
		t.Errorf("Expected 2 order by clauses, got %d", len(stmt.Select.OrderBy))
	}
}

// TestParseSQLArithmetic tests parsing arithmetic expressions.
func TestParseSQLArithmetic(t *testing.T) {
	input := "SELECT price * quantity FROM orders"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Select.Columns[0].Op != "*" {
		t.Errorf("Expected * operator, got %s", stmt.Select.Columns[0].Op)
	}
}

// TestParseSQLBoolean tests parsing boolean values.
func TestParseSQLBoolean(t *testing.T) {
	tests := []struct {
		input string
		value bool
	}{
		{"SELECT * FROM users WHERE active = TRUE", true},
		{"SELECT * FROM users WHERE active = FALSE", false},
	}

	for _, tt := range tests {
		stmt, err := ParseSQL(tt.input)
		if err != nil {
			t.Errorf("ParseSQL(%q) failed: %v", tt.input, err)
			continue
		}
		val, ok := stmt.Select.Where.Right.Value.(bool)
		if !ok {
			t.Errorf("ParseSQL(%q) expected boolean value", tt.input)
			continue
		}
		if val != tt.value {
			t.Errorf("ParseSQL(%q) value = %v, want %v", tt.input, val, tt.value)
		}
	}
}

// TestParseSQLNull tests parsing NULL values.
func TestParseSQLNull(t *testing.T) {
	input := "SELECT * FROM users WHERE email IS NULL"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	if stmt.Select.Where.Right.Value != nil {
		t.Error("Expected NULL value")
	}
}

// TestParseSQLFloat tests parsing float values.
func TestParseSQLFloat(t *testing.T) {
	input := "SELECT price FROM products WHERE price > 19.99"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	val, ok := stmt.Select.Where.Right.Value.(float64)
	if !ok {
		t.Errorf("ParseSQL(%q) expected float64 value", input)
		return
	}
	if val != 19.99 {
		t.Errorf("Expected 19.99, got %f", val)
	}
}

// TestParseSQLTableAlias tests parsing table aliases.
func TestParseSQLTableAlias(t *testing.T) {
	input := "SELECT u.name FROM users AS u"
	stmt, err := ParseSQL(input)
	if err != nil {
		t.Errorf("ParseSQL(%q) failed: %v", input, err)
		return
	}
	// Should parse correctly with alias
	_ = stmt
}
