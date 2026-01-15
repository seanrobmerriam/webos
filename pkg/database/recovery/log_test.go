package recovery

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewWAL tests creating a new WAL.
func TestNewWAL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	wal, err := NewWAL(path, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	if wal == nil {
		t.Fatal("WAL is nil")
	}
}

// TestWALWriteRead tests writing and reading log entries.
func TestWALWriteRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	wal, err := NewWAL(path, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Write some entries
	entries := []*LogEntry{
		{TxID: 1, Operation: OpBegin, TableName: "users"},
		{TxID: 1, Operation: OpInsert, TableName: "users", RowID: 100, AfterImage: []byte("test data")},
		{TxID: 1, Operation: OpCommit},
	}

	for _, entry := range entries {
		if err := wal.Write(entry); err != nil {
			t.Fatalf("Failed to write entry: %v", err)
		}
	}

	// Read entries back
	readEntries, err := wal.Read()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(readEntries))
	}

	// Verify entries
	for i, expected := range entries {
		actual := readEntries[i]
		if actual.TxID != expected.TxID {
			t.Errorf("Entry %d: expected TxID %d, got %d", i, expected.TxID, actual.TxID)
		}
		if actual.Operation != expected.Operation {
			t.Errorf("Entry %d: expected operation %v, got %v", i, expected.Operation, actual.Operation)
		}
		if actual.TableName != expected.TableName {
			t.Errorf("Entry %d: expected table name %s, got %s", i, expected.TableName, actual.TableName)
		}
	}
}

// TestWALGetLsn tests getting the current LSN.
func TestWALGetLsn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	wal, err := NewWAL(path, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Initial LSN should be 0
	if lsn := wal.GetLsn(); lsn != 0 {
		t.Errorf("Expected initial LSN 0, got %d", lsn)
	}

	// Write an entry
	entry := &LogEntry{TxID: 1, Operation: OpBegin}
	if err := wal.Write(entry); err != nil {
		t.Fatalf("Failed to write entry: %v", err)
	}

	// LSN should be 1
	if lsn := wal.GetLsn(); lsn != 1 {
		t.Errorf("Expected LSN 1, got %d", lsn)
	}
}

// TestWALClose tests closing the WAL.
func TestWALClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	wal, err := NewWAL(path, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Close the WAL
	if err := wal.Close(); err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}

	// Try to write after close - should fail
	entry := &LogEntry{TxID: 1, Operation: OpBegin}
	if err := wal.Write(entry); err != ErrWALClosed {
		t.Errorf("Expected ErrWALClosed, got %v", err)
	}
}

// TestLogEntrySerializeDeserialize tests serializing and deserializing log entries.
func TestLogEntrySerializeDeserialize(t *testing.T) {
	original := &LogEntry{
		TxID:        123,
		Operation:   OpInsert,
		TableName:   "test_table",
		RowID:       456,
		BeforeImage: []byte("before"),
		AfterImage:  []byte("after"),
		Timestamp:   789,
		LSN:         100,
	}

	// Serialize
	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Deserialize
	restored := &LogEntry{}
	if err := restored.Deserialize(data); err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify
	if restored.TxID != original.TxID {
		t.Errorf("TxID: expected %d, got %d", original.TxID, restored.TxID)
	}
	if restored.Operation != original.Operation {
		t.Errorf("Operation: expected %v, got %v", original.Operation, restored.Operation)
	}
	if restored.TableName != original.TableName {
		t.Errorf("TableName: expected %s, got %s", original.TableName, restored.TableName)
	}
	if restored.RowID != original.RowID {
		t.Errorf("RowID: expected %d, got %d", original.RowID, restored.RowID)
	}
	if string(restored.BeforeImage) != string(original.BeforeImage) {
		t.Errorf("BeforeImage: expected %s, got %s", original.BeforeImage, restored.BeforeImage)
	}
	if string(restored.AfterImage) != string(original.AfterImage) {
		t.Errorf("AfterImage: expected %s, got %s", original.AfterImage, restored.AfterImage)
	}
	if restored.Timestamp != original.Timestamp {
		t.Errorf("Timestamp: expected %d, got %d", original.Timestamp, restored.Timestamp)
	}
	if restored.LSN != original.LSN {
		t.Errorf("LSN: expected %d, got %d", original.LSN, restored.LSN)
	}
}

// TestWALMultipleTransactions tests writing multiple transactions.
func TestWALMultipleTransactions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	wal, err := NewWAL(path, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Transaction 1
	if err := wal.Write(&LogEntry{TxID: 1, Operation: OpBegin}); err != nil {
		t.Fatalf("Failed to write BEGIN: %v", err)
	}
	if err := wal.Write(&LogEntry{TxID: 1, Operation: OpInsert, TableName: "users", RowID: 1, AfterImage: []byte("user1")}); err != nil {
		t.Fatalf("Failed to write INSERT: %v", err)
	}
	if err := wal.Write(&LogEntry{TxID: 1, Operation: OpCommit}); err != nil {
		t.Fatalf("Failed to write COMMIT: %v", err)
	}

	// Transaction 2
	if err := wal.Write(&LogEntry{TxID: 2, Operation: OpBegin}); err != nil {
		t.Fatalf("Failed to write BEGIN: %v", err)
	}
	if err := wal.Write(&LogEntry{TxID: 2, Operation: OpInsert, TableName: "users", RowID: 2, AfterImage: []byte("user2")}); err != nil {
		t.Fatalf("Failed to write INSERT: %v", err)
	}
	if err := wal.Write(&LogEntry{TxID: 2, Operation: OpRollback}); err != nil {
		t.Fatalf("Failed to write ROLLBACK: %v", err)
	}

	// Read entries
	entries, err := wal.Read()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(entries) != 6 {
		t.Fatalf("Expected 6 entries, got %d", len(entries))
	}

	// Verify transaction boundaries
	if entries[0].Operation != OpBegin || entries[0].TxID != 1 {
		t.Error("First entry should be BEGIN for tx 1")
	}
	if entries[2].Operation != OpCommit || entries[2].TxID != 1 {
		t.Error("Third entry should be COMMIT for tx 1")
	}
	if entries[3].Operation != OpBegin || entries[3].TxID != 2 {
		t.Error("Fourth entry should be BEGIN for tx 2")
	}
	if entries[5].Operation != OpRollback || entries[5].TxID != 2 {
		t.Error("Sixth entry should be ROLLBACK for tx 2")
	}
}

// TestWALCorrupted tests handling of corrupted WAL.
func TestWALCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	// Create a WAL file with corrupted data
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	// Write some invalid data
	f.Write([]byte{0, 0, 0, 10, 1, 2, 3, 4, 5, 6, 7, 8})
	f.Close()

	wal, err := NewWAL(path, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Read should fail with corrupted entry
	_, err = wal.Read()
	if err == nil {
		t.Error("Expected error for corrupted WAL")
	}
}

// TestWALEmptyRead tests reading an empty WAL.
func TestWALEmptyRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	wal, err := NewWAL(path, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	entries, err := wal.Read()
	if err != nil {
		t.Fatalf("Failed to read empty WAL: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

// TestLogOperationString tests string representation of log operations.
func TestLogOperationString(t *testing.T) {
	tests := []struct {
		op   LogOperation
		want string
	}{
		{OpBegin, "BEGIN"},
		{OpInsert, "INSERT"},
		{OpUpdate, "UPDATE"},
		{OpDelete, "DELETE"},
		{OpCommit, "COMMIT"},
		{OpRollback, "ROLLBACK"},
		{OpCheckpoint, "CHECKPOINT"},
		{255, "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("LogOperation(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}
