// Database demo program demonstrating the SQL database engine.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"webos/pkg/database"
	"webos/pkg/database/query"
	"webos/pkg/database/recovery"
	"webos/pkg/database/txn"
)

func main() {
	fmt.Println("=== WebOS Database Engine Demo ===")
	fmt.Println()

	// Create a temporary directory for the demo
	tempDir, err := os.MkdirTemp("", "webos-db-demo-*")
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	// Create a new database
	db, err := database.NewDatabase("demo", tempDir)
	if err != nil {
		fmt.Printf("Error creating database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("1. Creating tables...")

	// Create a users table
	usersSchema := &database.Schema{
		TableName: "users",
		Columns: []database.ColumnDefinition{
			{Name: "id", Type: database.DataTypeInteger, PrimaryKey: true},
			{Name: "name", Type: database.DataTypeText, NotNull: true},
			{Name: "email", Type: database.DataTypeText},
			{Name: "age", Type: database.DataTypeInteger},
		},
		PrimaryKey: []string{"id"},
	}

	usersTable, err := db.CreateTable("users", usersSchema)
	if err != nil {
		fmt.Printf("Error creating users table: %v\n", err)
		os.Exit(1)
	}

	// Create an orders table
	ordersSchema := &database.Schema{
		TableName: "orders",
		Columns: []database.ColumnDefinition{
			{Name: "id", Type: database.DataTypeInteger, PrimaryKey: true},
			{Name: "user_id", Type: database.DataTypeInteger, NotNull: true},
			{Name: "amount", Type: database.DataTypeFloat},
			{Name: "status", Type: database.DataTypeText},
		},
		PrimaryKey: []string{"id"},
	}

	ordersTable, err := db.CreateTable("orders", ordersSchema)
	if err != nil {
		fmt.Printf("Error creating orders table: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("   - Created 'users' table")
	fmt.Println("   - Created 'orders' table")

	fmt.Println("\n2. Inserting data...")

	// Insert some users using NewValue
	val1, _ := database.NewValue(int64(1))
	val2, _ := database.NewValue("Alice")
	val3, _ := database.NewValue("alice@example.com")
	val4, _ := database.NewValue(int64(30))
	_, err = usersTable.Insert([]database.Value{val1, val2, val3, val4})
	if err != nil {
		fmt.Printf("Error inserting user1: %v\n", err)
	}
	fmt.Println("   - Inserted user 1: Alice")

	val5, _ := database.NewValue(int64(2))
	val6, _ := database.NewValue("Bob")
	val7, _ := database.NewValue("bob@example.com")
	val8, _ := database.NewValue(int64(25))
	_, err = usersTable.Insert([]database.Value{val5, val6, val7, val8})
	if err != nil {
		fmt.Printf("Error inserting user2: %v\n", err)
	}
	fmt.Println("   - Inserted user 2: Bob")

	val9, _ := database.NewValue(int64(3))
	val10, _ := database.NewValue("Charlie")
	val11, _ := database.NewValue("charlie@example.com")
	val12, _ := database.NewValue(int64(35))
	_, err = usersTable.Insert([]database.Value{val9, val10, val11, val12})
	if err != nil {
		fmt.Printf("Error inserting user3: %v\n", err)
	}
	fmt.Println("   - Inserted user 3: Charlie")

	// Insert some orders
	oval1, _ := database.NewValue(int64(1))
	oval2, _ := database.NewValue(int64(1))
	oval3, _ := database.NewValue(99.99)
	oval4, _ := database.NewValue("completed")
	_, err = ordersTable.Insert([]database.Value{oval1, oval2, oval3, oval4})
	if err != nil {
		fmt.Printf("Error inserting order1: %v\n", err)
	}
	fmt.Println("   - Inserted order 1")

	oval5, _ := database.NewValue(int64(2))
	oval6, _ := database.NewValue(int64(1))
	oval7, _ := database.NewValue(149.99)
	oval8, _ := database.NewValue("completed")
	_, err = ordersTable.Insert([]database.Value{oval5, oval6, oval7, oval8})
	if err != nil {
		fmt.Printf("Error inserting order2: %v\n", err)
	}
	fmt.Println("   - Inserted order 2")

	oval9, _ := database.NewValue(int64(3))
	oval10, _ := database.NewValue(int64(2))
	oval11, _ := database.NewValue(49.99)
	oval12, _ := database.NewValue("pending")
	_, err = ordersTable.Insert([]database.Value{oval9, oval10, oval11, oval12})
	if err != nil {
		fmt.Printf("Error inserting order3: %v\n", err)
	}
	fmt.Println("   - Inserted order 3")

	fmt.Println("\n3. Querying data...")

	// Demo SQL parsing
	fmt.Println("\n   SQL Parser Demo:")
	fmt.Println("   ----------------")

	// Parse a SELECT statement
	sql := "SELECT id, name, email FROM users WHERE age > 25"
	stmt, err := query.ParseSQL(sql)
	if err != nil {
		fmt.Printf("   Error parsing SQL: %v\n", err)
	} else {
		fmt.Printf("   Parsed: %s\n", sql)
		fmt.Printf("   Statement type: %v\n", stmt.Type)
	}

	// Parse an INSERT statement
	sql = "INSERT INTO users (name, email) VALUES ('David', 'david@example.com')"
	stmt, err = query.ParseSQL(sql)
	if err != nil {
		fmt.Printf("   Error parsing SQL: %v\n", err)
	} else {
		fmt.Printf("   Parsed: %s\n", sql)
		fmt.Printf("   Statement type: %v\n", stmt.Type)
	}

	// Parse an UPDATE statement
	sql = "UPDATE users SET age = 31 WHERE name = 'Alice'"
	stmt, err = query.ParseSQL(sql)
	if err != nil {
		fmt.Printf("   Error parsing SQL: %v\n", err)
	} else {
		fmt.Printf("   Parsed: %s\n", sql)
		fmt.Printf("   Statement type: %v\n", stmt.Type)
	}

	// Parse a DELETE statement
	sql = "DELETE FROM users WHERE age < 30"
	stmt, err = query.ParseSQL(sql)
	if err != nil {
		fmt.Printf("   Error parsing SQL: %v\n", err)
	} else {
		fmt.Printf("   Parsed: %s\n", sql)
		fmt.Printf("   Statement type: %v\n", stmt.Type)
	}

	fmt.Println("\n4. Reading table data...")

	// Read all users
	fmt.Println("\n   Users table:")
	users, _ := usersTable.Select(nil)
	for i, user := range users {
		fmt.Printf("   Row %d: ID=%d, Name=%s, Email=%s, Age=%d\n",
			i, user.Values[0].Int, user.Values[1].Str, user.Values[2].Str, user.Values[3].Int)
	}

	// Read all orders
	fmt.Println("\n   Orders table:")
	orders, _ := ordersTable.Select(nil)
	for i, order := range orders {
		fmt.Printf("   Row %d: ID=%d, UserID=%d, Amount=%.2f, Status=%s\n",
			i, order.Values[0].Int, order.Values[1].Int, order.Values[2].Float, order.Values[3].Str)
	}

	fmt.Println("\n5. Testing B-tree index...")

	// Create an index on users table
	if err := usersTable.CreateIndex("idx_users_name", []string{"name"}, false); err != nil {
		fmt.Printf("   Error creating index: %v\n", err)
	} else {
		fmt.Println("   - Created index 'idx_users_name' on 'name' column")
	}

	fmt.Println("\n6. Testing transaction manager...")

	// Test transaction operations
	txnMgr := txn.NewTransactionManager(10, txn.IsolationReadCommitted)
	txn, err := txnMgr.Begin()
	if err != nil {
		fmt.Printf("   Error starting transaction: %v\n", err)
	} else {
		fmt.Printf("   - Started transaction %d\n", txn.ID)

		if err := txnMgr.Commit(txn.ID); err != nil {
			fmt.Printf("   Error committing transaction: %v\n", err)
		} else {
			fmt.Println("   - Committed transaction")
		}
	}

	fmt.Println("\n7. Testing WAL recovery...")

	// Create a WAL for testing
	walPath := filepath.Join(tempDir, "test.wal")
	wal, err := recovery.NewWAL(walPath, 1024*1024)
	if err != nil {
		fmt.Printf("   Error creating WAL: %v\n", err)
	} else {
		// Write some log entries
		err := wal.Write(&recovery.LogEntry{
			TxID:      1,
			Operation: recovery.OpBegin,
			TableName: "users",
		})
		if err != nil {
			fmt.Printf("   Error writing to WAL: %v\n", err)
		}

		err = wal.Write(&recovery.LogEntry{
			TxID:      1,
			Operation: recovery.OpCommit,
			TableName: "users",
		})
		if err != nil {
			fmt.Printf("   Error writing to WAL: %v\n", err)
		}

		// Read entries back
		entries, err := wal.Read()
		if err != nil {
			fmt.Printf("   Error reading WAL: %v\n", err)
		} else {
			fmt.Printf("   - Read %d entries from WAL\n", len(entries))
		}

		wal.Close()
		fmt.Println("   - Closed WAL")
	}

	fmt.Println("\n8. Saving database...")

	// Save the database
	if err := db.Save(); err != nil {
		fmt.Printf("   Error saving database: %v\n", err)
	} else {
		fmt.Println("   - Database saved successfully")
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nSummary:")
	fmt.Println("- Created and populated 'users' and 'orders' tables")
	fmt.Println("- Demonstrated SQL parsing for SELECT, INSERT, UPDATE, DELETE")
	fmt.Println("- Tested B-tree index creation")
	fmt.Println("- Tested transaction Begin/Commit operations")
	fmt.Println("- Tested WAL write/read operations")
	fmt.Println("- Database saved to:", tempDir)
}
