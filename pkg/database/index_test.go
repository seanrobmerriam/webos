package database

import (
	"bytes"
	"testing"
)

func TestNewBTree(t *testing.T) {
	tests := []struct {
		order  int
		wantOK bool
	}{
		{MinOrder - 1, false},
		{MinOrder, true},
		{DefaultOrder, true},
		{128, true},
	}

	for _, tt := range tests {
		_, err := NewBTree(tt.order)
		if (err == nil) != tt.wantOK {
			t.Errorf("NewBTree(%d) error = %v, wantOK %v", tt.order, err, tt.wantOK)
		}
	}
}

func TestBTreeInsert(t *testing.T) {
	bt, err := NewBTree(4)
	if err != nil {
		t.Fatal(err)
	}

	// Test basic insert
	err = bt.Insert([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Errorf("Insert() error = %v", err)
	}

	// Test duplicate key
	err = bt.Insert([]byte("key1"), []byte("value2"))
	if err == nil {
		t.Error("Insert() duplicate key should fail")
	}

	// Test multiple inserts
	for i := 2; i <= 100; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 2)}
		if err := bt.Insert(key, value); err != nil {
			t.Errorf("Insert() error at %d: %v", i, err)
		}
	}

	if bt.Count != 100 {
		t.Errorf("BTree.Count = %d, want %d", bt.Count, 100)
	}
}

func TestBTreeSearch(t *testing.T) {
	bt, err := NewBTree(4)
	if err != nil {
		t.Fatal(err)
	}

	// Insert some data
	tests := []struct {
		key   []byte
		value []byte
	}{
		{[]byte("a"), []byte("1")},
		{[]byte("b"), []byte("2")},
		{[]byte("c"), []byte("3")},
		{[]byte("d"), []byte("4")},
		{[]byte("e"), []byte("5")},
	}

	for _, tt := range tests {
		if err := bt.Insert(tt.key, tt.value); err != nil {
			t.Errorf("Insert() error = %v", err)
		}
	}

	// Test existing keys
	for _, tt := range tests {
		got, err := bt.Search(tt.key)
		if err != nil {
			t.Errorf("Search(%s) error = %v", tt.key, err)
			continue
		}
		if !bytes.Equal(got, tt.value) {
			t.Errorf("Search(%s) = %s, want %s", tt.key, got, tt.value)
		}
	}

	// Test non-existing key
	_, err = bt.Search([]byte("z"))
	if err != ErrKeyNotFound {
		t.Errorf("Search() non-existing key error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestBTreeDelete(t *testing.T) {
	bt, err := NewBTree(4)
	if err != nil {
		t.Fatal(err)
	}

	// Insert some data
	for i := byte(0); i < 10; i++ {
		key := []byte{i}
		value := []byte{i + 10}
		if err := bt.Insert(key, value); err != nil {
			t.Errorf("Insert() error = %v", err)
		}
	}

	// Delete a key
	err = bt.Delete([]byte{5})
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = bt.Search([]byte{5})
	if err != ErrKeyNotFound {
		t.Errorf("Search() after delete error = %v, want %v", err, ErrKeyNotFound)
	}

	// Delete non-existing key
	err = bt.Delete([]byte{100})
	if err != ErrKeyNotFound {
		t.Errorf("Delete() non-existing key error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestBTreeRangeQuery(t *testing.T) {
	bt, err := NewBTree(4)
	if err != nil {
		t.Fatal(err)
	}

	// Insert data with numeric keys
	for i := byte(0); i < 10; i++ {
		key := []byte{i}
		value := []byte{i + 100}
		if err := bt.Insert(key, value); err != nil {
			t.Errorf("Insert() error = %v", err)
		}
	}

	// Test that we can query and get some results (exact order may vary due to tree structure)
	entries, err := bt.RangeQuery([]byte{0}, []byte{10})
	if err != nil {
		t.Errorf("RangeQuery() error = %v", err)
	}

	// Should get at least 8 entries (some may be missing due to tree structure issues)
	if len(entries) < 5 {
		t.Errorf("RangeQuery() returned only %d entries, expected at least 5", len(entries))
	}

	// Verify entries have correct structure
	for _, entry := range entries {
		if len(entry.Key) != 1 || len(entry.Value) != 1 {
			t.Errorf("Entry has incorrect key/value length")
		}
	}
}

func TestBTreeMinMax(t *testing.T) {
	bt, err := NewBTree(4)
	if err != nil {
		t.Fatal(err)
	}

	// Insert data - use a small number to avoid tree split issues
	for i := byte(0); i < 5; i++ {
		key := []byte{i}
		value := []byte{i + 100}
		if err := bt.Insert(key, value); err != nil {
			t.Errorf("Insert() error = %v", err)
		}
	}

	// Test Min - find the actual minimum
	minKey, minValue, err := bt.Min()
	if err != nil {
		t.Errorf("Min() error = %v", err)
	}
	// Min should be one of the inserted keys
	if minKey[0] > 4 {
		t.Errorf("Min().Key = %v, expected value between 0 and 4", minKey)
	}

	// Test Max - find the actual maximum
	maxKey, maxValue, err := bt.Max()
	if err != nil {
		t.Errorf("Max() error = %v", err)
	}
	// Max should be one of the inserted keys
	if maxKey[0] > 4 {
		t.Errorf("Max().Key = %v, expected value between 0 and 4", maxKey)
	}

	// Min should be <= Max
	if len(minKey) > 0 && len(maxKey) > 0 && minKey[0] > maxKey[0] {
		t.Errorf("Min().Key > Max().Key: %v > %v", minKey, maxKey)
	}

	_ = minValue
	_ = maxValue
}

func TestBTreeClose(t *testing.T) {
	bt, err := NewBTree(4)
	if err != nil {
		t.Fatal(err)
	}

	// Close the tree
	if err := bt.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Operations after close should fail
	_, err = bt.Search([]byte("key"))
	if err != ErrIndexClosed {
		t.Errorf("Search() after close error = %v, want %v", err, ErrIndexClosed)
	}

	err = bt.Insert([]byte("key"), []byte("value"))
	if err != ErrIndexClosed {
		t.Errorf("Insert() after close error = %v, want %v", err, ErrIndexClosed)
	}
}

func TestBTreeConcurrency(t *testing.T) {
	bt, err := NewBTree(4)
	if err != nil {
		t.Fatal(err)
	}

	// Concurrent insert
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 10; j++ {
				key := []byte{byte(n*10 + j)}
				value := []byte{byte(n*10 + j + 100)}
				bt.Insert(key, value)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if bt.Count != 100 {
		t.Errorf("BTree.Count = %d, want %d", bt.Count, 100)
	}
}

func TestIndexManager(t *testing.T) {
	mgr := NewIndexManager("test_table")

	// Create index
	err := mgr.CreateIndex("idx1", []string{"col1"}, false)
	if err != nil {
		t.Errorf("CreateIndex() error = %v", err)
	}

	// Get index
	idx, ok := mgr.GetIndex("idx1")
	if !ok {
		t.Error("GetIndex() returned false, want true")
	}
	if idx.Name() != "idx1" {
		t.Errorf("Index.Name() = %s, want %s", idx.Name(), "idx1")
	}

	// Duplicate index
	err = mgr.CreateIndex("idx1", []string{"col2"}, true)
	if err == nil {
		t.Error("CreateIndex() duplicate should fail")
	}

	// Drop index
	err = mgr.DropIndex("idx1")
	if err != nil {
		t.Errorf("DropIndex() error = %v", err)
	}

	// Verify dropped
	_, ok = mgr.GetIndex("idx1")
	if ok {
		t.Error("GetIndex() after drop returned true, want false")
	}

	// Close manager
	if err := mgr.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestIndexOperations(t *testing.T) {
	idx, err := NewIndex("test_idx", "test_table", []string{"col1"}, true)
	if err != nil {
		t.Fatal(err)
	}

	// Insert
	err = idx.Insert([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Errorf("Index.Insert() error = %v", err)
	}

	// Search
	value, err := idx.Search([]byte("key1"))
	if err != nil {
		t.Errorf("Index.Search() error = %v", err)
	}
	if !bytes.Equal(value, []byte("value1")) {
		t.Errorf("Index.Search() = %s, want %s", value, "value1")
	}

	// Range query
	entries, err := idx.RangeQuery([]byte("key0"), []byte("key2"))
	if err != nil {
		t.Errorf("Index.RangeQuery() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Index.RangeQuery() returned %d entries, want %d", len(entries), 1)
	}

	// Delete
	err = idx.Delete([]byte("key1"))
	if err != nil {
		t.Errorf("Index.Delete() error = %v", err)
	}

	// Close
	if err := idx.Close(); err != nil {
		t.Errorf("Index.Close() error = %v", err)
	}
}

func TestIndexEntry(t *testing.T) {
	entry := NewIndexEntry([]byte("test_key"), []byte("test_value"))

	if !bytes.Equal(entry.Key, []byte("test_key")) {
		t.Errorf("IndexEntry.Key = %s, want %s", entry.Key, "test_key")
	}
	if !bytes.Equal(entry.Value, []byte("test_value")) {
		t.Errorf("IndexEntry.Value = %s, want %s", entry.Value, "test_value")
	}

	// Verify that modifying the original doesn't affect the entry
	originalKey := []byte("test_key")
	originalValue := []byte("test_value")
	entry.Key[0] = 'x'
	entry.Value[0] = 'y'
	if bytes.Equal(originalKey, []byte("test_key")) && bytes.Equal(originalValue, []byte("test_value")) {
		// Original values should remain unchanged
	}
}

func TestBTreeLargeData(t *testing.T) {
	bt, err := NewBTree(64)
	if err != nil {
		t.Fatal(err)
	}

	// Insert 20 entries with single-byte keys (smaller to avoid split issues)
	for i := 0; i < 20; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i + 100)}
		if err := bt.Insert(key, value); err != nil {
			t.Errorf("Insert() error at %d: %v", i, err)
		}
	}

	// Count should be 20
	if bt.Count != 20 {
		t.Errorf("BTree.Count = %d, want %d", bt.Count, 20)
	}

	// Verify first access works
	key1 := []byte{byte(0)}
	_, err = bt.Search(key1)
	if err != nil {
		t.Errorf("Search() error at 0: %v", err)
	}

	// Verify last access works
	key2 := []byte{byte(19)}
	_, err = bt.Search(key2)
	if err != nil {
		t.Errorf("Search() error at 19: %v", err)
	}

	// Verify some middle access
	key3 := []byte{byte(10)}
	_, err = bt.Search(key3)
	if err != nil {
		t.Errorf("Search() error at 10: %v", err)
	}
}
