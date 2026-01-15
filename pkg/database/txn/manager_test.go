package txn

import (
	"testing"
	"time"
)

// TestNewTransactionManager tests creating a new transaction manager.
func TestNewTransactionManager(t *testing.T) {
	mgr := NewTransactionManager(100, IsolationReadCommitted)
	if mgr == nil {
		t.Fatal("TransactionManager is nil")
	}
	if mgr.maxActiveTxns != 100 {
		t.Errorf("Expected maxActiveTxns 100, got %d", mgr.maxActiveTxns)
	}
	if mgr.GetIsolationLevel() != IsolationReadCommitted {
		t.Errorf("Expected isolation level READ COMMITTED, got %v", mgr.GetIsolationLevel())
	}
}

// TestTransactionManagerBegin tests starting new transactions.
func TestTransactionManagerBegin(t *testing.T) {
	mgr := NewTransactionManager(10, IsolationReadCommitted)

	txn1, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	if txn1.ID != 1 {
		t.Errorf("Expected transaction ID 1, got %d", txn1.ID)
	}
	if !txn1.IsActive() {
		t.Error("Transaction should be active")
	}

	txn2, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin second transaction: %v", err)
	}
	if txn2.ID != 2 {
		t.Errorf("Expected transaction ID 2, got %d", txn2.ID)
	}

	if mgr.ActiveTransactions() != 2 {
		t.Errorf("Expected 2 active transactions, got %d", mgr.ActiveTransactions())
	}
}

// TestTransactionManagerCommit tests committing transactions.
func TestTransactionManagerCommit(t *testing.T) {
	mgr := NewTransactionManager(10, IsolationReadCommitted)

	txn, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Commit the transaction
	if err := mgr.Commit(txn.ID); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	if mgr.ActiveTransactions() != 0 {
		t.Errorf("Expected 0 active transactions, got %d", mgr.ActiveTransactions())
	}

	// Transaction should no longer be in the manager
	_, ok := mgr.Get(txn.ID)
	if ok {
		t.Error("Transaction should not be found after commit")
	}

	// Transaction state should be committed
	if !txn.IsCommitted() {
		t.Error("Transaction should be committed")
	}
}

// TestTransactionManagerRollback tests rolling back transactions.
func TestTransactionManagerRollback(t *testing.T) {
	mgr := NewTransactionManager(10, IsolationReadCommitted)

	txn, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Rollback the transaction
	if err := mgr.Rollback(txn.ID); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	if mgr.ActiveTransactions() != 0 {
		t.Errorf("Expected 0 active transactions, got %d", mgr.ActiveTransactions())
	}

	// Transaction should no longer be in the manager
	_, ok := mgr.Get(txn.ID)
	if ok {
		t.Error("Transaction should not be found after rollback")
	}

	// Transaction state should be rolled back
	if !txn.IsRolledBack() {
		t.Error("Transaction should be rolled back")
	}
}

// TestTransactionManagerCommitTwice tests committing a transaction twice.
func TestTransactionManagerCommitTwice(t *testing.T) {
	mgr := NewTransactionManager(10, IsolationReadCommitted)

	txn, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// First commit should succeed
	if err := mgr.Commit(txn.ID); err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	// Second commit should fail
	if err := mgr.Commit(txn.ID); err != ErrTransactionCommitted {
		t.Errorf("Expected ErrTransactionCommitted, got %v", err)
	}
}

// TestTransactionManagerRollbackTwice tests rolling back a transaction twice.
func TestTransactionManagerRollbackTwice(t *testing.T) {
	mgr := NewTransactionManager(10, IsolationReadCommitted)

	txn, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// First rollback should succeed
	if err := mgr.Rollback(txn.ID); err != nil {
		t.Fatalf("First rollback failed: %v", err)
	}

	// Second rollback should fail
	if err := mgr.Rollback(txn.ID); err != ErrTransactionRolledBack {
		t.Errorf("Expected ErrTransactionRolledBack, got %v", err)
	}
}

// TestTransactionManagerCommitAfterRollback tests committing after rollback.
func TestTransactionManagerCommitAfterRollback(t *testing.T) {
	mgr := NewTransactionManager(10, IsolationReadCommitted)

	txn, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Rollback first
	if err := mgr.Rollback(txn.ID); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Commit should fail
	if err := mgr.Commit(txn.ID); err != ErrTransactionRolledBack {
		t.Errorf("Expected ErrTransactionRolledBack, got %v", err)
	}
}

// TestTransactionManagerMaxTransactions tests limiting concurrent transactions.
func TestTransactionManagerMaxTransactions(t *testing.T) {
	mgr := NewTransactionManager(3, IsolationReadCommitted)

	// Start 3 transactions
	for i := 0; i < 3; i++ {
		_, err := mgr.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction %d: %v", i+1, err)
		}
	}

	// Fourth transaction should fail
	_, err := mgr.Begin()
	if err != ErrTooManyTransactions {
		t.Errorf("Expected ErrTooManyTransactions, got %v", err)
	}
}

// TestTransactionRecordBeforeImage tests recording before images.
func TestTransactionRecordBeforeImage(t *testing.T) {
	txn := &Transaction{
		ID:           1,
		modifiedRows: make(map[uint64][]byte),
	}

	beforeImage := []byte("old data")
	txn.RecordBeforeImage(100, beforeImage)

	result := txn.GetBeforeImage(100)
	if string(result) != "old data" {
		t.Errorf("Expected 'old data', got '%s'", string(result))
	}
}

// TestTransactionMarkTableModified tests marking tables as modified.
func TestTransactionMarkTableModified(t *testing.T) {
	txn := &Transaction{
		modifiedTables: make(map[string]bool),
	}

	txn.MarkTableModified("users")
	txn.MarkTableModified("orders")

	if !txn.IsTableModified("users") {
		t.Error("users should be marked as modified")
	}
	if !txn.IsTableModified("orders") {
		t.Error("orders should be marked as modified")
	}
	if txn.IsTableModified("products") {
		t.Error("products should not be marked as modified")
	}
}

// TestTransactionGetModifiedTables tests getting modified tables.
func TestTransactionGetModifiedTables(t *testing.T) {
	txn := &Transaction{
		modifiedTables: make(map[string]bool),
	}

	txn.MarkTableModified("users")
	txn.MarkTableModified("orders")

	tables := txn.GetModifiedTables()
	if len(tables) != 2 {
		t.Errorf("Expected 2 modified tables, got %d", len(tables))
	}
}

// TestIsolationLevelString tests string representation of isolation levels.
func TestIsolationLevelString(t *testing.T) {
	tests := []struct {
		level IsolationLevel
		want  string
	}{
		{IsolationReadCommitted, "READ COMMITTED"},
		{IsolationRepeatableRead, "REPEATABLE READ"},
		{IsolationSerializable, "SERIALIZABLE"},
		{255, "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("IsolationLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

// TestTransactionProperties tests transaction properties.
func TestTransactionProperties(t *testing.T) {
	mgr := NewTransactionManager(10, IsolationSerializable)

	txn, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	if txn.ID != 1 {
		t.Errorf("Expected ID 1, got %d", txn.ID)
	}
	if txn.State != StateActive {
		t.Errorf("Expected state Active, got %d", txn.State)
	}
	if txn.Isolation != IsolationSerializable {
		t.Errorf("Expected isolation SERIALIZABLE, got %v", txn.Isolation)
	}
	if time.Since(txn.StartTime) > time.Second {
		t.Error("StartTime should be recent")
	}
}

// TestTransactionManagerSetIsolationLevel tests setting isolation level.
func TestTransactionManagerSetIsolationLevel(t *testing.T) {
	mgr := NewTransactionManager(10, IsolationReadCommitted)

	mgr.SetIsolationLevel(IsolationSerializable)
	if mgr.GetIsolationLevel() != IsolationSerializable {
		t.Error("Isolation level should be SERIALIZABLE")
	}

	// New transactions should use the new isolation level
	txn, err := mgr.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	if txn.Isolation != IsolationSerializable {
		t.Errorf("Transaction should have SERIALIZABLE isolation, got %v", txn.Isolation)
	}
}
