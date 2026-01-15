// Package database provides a SQL database engine with ACID transactions.
package database

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
)

const (
	// DefaultOrder is the default B-tree order (maximum children per node).
	DefaultOrder = 64
	// MinOrder is the minimum B-tree order.
	MinOrder = 4
)

// Index errors.
var (
	// ErrKeyNotFound indicates the key was not found in the index.
	ErrKeyNotFound = errors.New("key not found")
	// ErrDuplicateKey indicates a duplicate key insertion.
	ErrDuplicateKey = errors.New("duplicate key")
	// ErrInvalidOrder indicates an invalid B-tree order.
	ErrInvalidOrder = errors.New("invalid B-tree order")
	// ErrIndexClosed indicates operations on a closed index.
	ErrIndexClosed = errors.New("index is closed")
)

// IndexEntry represents a key-value pair in the index.
type IndexEntry struct {
	Key   []byte // Key bytes
	Value []byte // Value bytes (row pointer)
}

// NewIndexEntry creates a new index entry.
func NewIndexEntry(key, value []byte) *IndexEntry {
	keyCopy := make([]byte, len(key))
	valueCopy := make([]byte, len(value))
	copy(keyCopy, key)
	copy(valueCopy, value)
	return &IndexEntry{Key: keyCopy, Value: valueCopy}
}

// BTreeNode represents a node in the B-tree.
type BTreeNode struct {
	Leaf     bool         // Whether this is a leaf node
	Keys     [][]byte     // Keys in this node
	Values   [][]byte     // Values in this node (leaf nodes only)
	Children []*BTreeNode // Child pointers (internal nodes only)
}

// BTree represents a B-tree index.
type BTree struct {
	Root   *BTreeNode // Root node
	Order  int        // Maximum children per node (fanout)
	Count  int        // Number of entries
	mu     sync.RWMutex
	closed bool
}

// NewBTree creates a new B-tree with the specified order.
func NewBTree(order int) (*BTree, error) {
	if order < MinOrder {
		return nil, fmt.Errorf("%w: order must be at least %d", ErrInvalidOrder, MinOrder)
	}
	return &BTree{
		Root:  &BTreeNode{Leaf: true, Keys: [][]byte{}, Values: [][]byte{}},
		Order: order,
		Count: 0,
	}, nil
}

// Search searches for a key in the B-tree.
func (t *BTree) Search(key []byte) ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, ErrIndexClosed
	}

	node, idx := t.search(t.Root, key)
	if node == nil {
		return nil, ErrKeyNotFound
	}
	return node.Values[idx], nil
}

// search searches for a key starting from the given node.
func (t *BTree) search(node *BTreeNode, key []byte) (*BTreeNode, int) {
	if node == nil {
		return nil, -1
	}

	idx := 0
	for idx < len(node.Keys) && bytes.Compare(node.Keys[idx], key) < 0 {
		idx++
	}

	if idx < len(node.Keys) && bytes.Equal(node.Keys[idx], key) {
		return node, idx
	}

	if node.Leaf {
		return nil, -1
	}

	return t.search(node.Children[idx], key)
}

// Insert inserts a key-value pair into the B-tree.
func (t *BTree) Insert(key, value []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrIndexClosed
	}

	// Check for duplicate key
	if t.containsKey(t.Root, key) {
		return fmt.Errorf("%w: %v", ErrDuplicateKey, key)
	}

	root := t.Root
	if len(root.Keys) == 2*t.Order-1 {
		// Root is full, need to split
		newRoot := &BTreeNode{Leaf: false, Keys: [][]byte{}, Children: []*BTreeNode{root}}
		t.splitChild(newRoot, 0)
		t.insertNonFull(newRoot, key, value)
		t.Root = newRoot
	} else {
		t.insertNonFull(root, key, value)
	}

	t.Count++
	return nil
}

// containsKey checks if a key exists in the subtree.
func (t *BTree) containsKey(node *BTreeNode, key []byte) bool {
	if node == nil {
		return false
	}

	idx := 0
	for idx < len(node.Keys) && bytes.Compare(node.Keys[idx], key) < 0 {
		idx++
	}

	if idx < len(node.Keys) && bytes.Equal(node.Keys[idx], key) {
		return true
	}

	if node.Leaf {
		return false
	}

	return t.containsKey(node.Children[idx], key)
}

// insertNonFull inserts a key into a node that is not full.
func (t *BTree) insertNonFull(node *BTreeNode, key, value []byte) {
	if node.Leaf {
		// Find the correct position to insert
		idx := 0
		for idx < len(node.Keys) && bytes.Compare(node.Keys[idx], key) < 0 {
			idx++
		}

		// Insert the key and value at the correct position
		node.Keys = append(node.Keys, nil)
		node.Values = append(node.Values, nil)
		copy(node.Keys[idx+1:], node.Keys[idx:])
		copy(node.Values[idx+1:], node.Values[idx:])

		node.Keys[idx] = make([]byte, len(key))
		node.Values[idx] = make([]byte, len(value))
		copy(node.Keys[idx], key)
		copy(node.Values[idx], value)
		return
	}

	// Internal node
	idx := 0
	for idx < len(node.Keys) && bytes.Compare(node.Keys[idx], key) > 0 {
		idx++
	}

	child := node.Children[idx]
	if len(child.Keys) == 2*t.Order-1 {
		t.splitChild(node, idx)
		if bytes.Compare(node.Keys[idx], key) < 0 {
			idx++
		}
	}

	t.insertNonFull(node.Children[idx], key, value)
}

// splitChild splits a full child node.
func (t *BTree) splitChild(parent *BTreeNode, index int) {
	child := parent.Children[index]

	// Create new sibling node
	sibling := &BTreeNode{Leaf: child.Leaf, Keys: [][]byte{}, Values: [][]byte{}}

	if !child.Leaf {
		sibling.Children = []*BTreeNode{}
	}

	// Number of keys to move to sibling
	numToMove := t.Order - 1

	// Copy the right half of keys and values to sibling
	sibling.Keys = make([][]byte, numToMove)
	sibling.Values = make([][]byte, numToMove)
	copy(sibling.Keys, child.Keys[t.Order:])
	copy(sibling.Values, child.Values[t.Order:])

	if !child.Leaf {
		sibling.Children = make([]*BTreeNode, numToMove+1)
		copy(sibling.Children, child.Children[t.Order:])
	}

	// Update child - keep the left half
	child.Keys = child.Keys[:t.Order]
	child.Values = child.Values[:t.Order]
	if !child.Leaf {
		child.Children = child.Children[:t.Order+1]
	}

	// Move the middle key up to parent
	middleKey := child.Keys[t.Order-1]

	// Insert middle key into parent
	parent.Keys = append(parent.Keys, nil)
	parent.Values = append(parent.Values, nil)
	copy(parent.Keys[index+1:], parent.Keys[index:])
	copy(parent.Values[index+1:], parent.Values[index:])
	parent.Keys[index] = middleKey
	parent.Values[index] = nil

	// Insert sibling into parent's children
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[index+1:], parent.Children[index:])
	parent.Children[index+1] = sibling
}

// Delete deletes a key from the B-tree.
func (t *BTree) Delete(key []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrIndexClosed
	}

	err := t.delete(t.Root, key)
	if err == nil {
		t.Count--
	}
	return err
}

// delete deletes a key from the B-tree.
func (t *BTree) delete(node *BTreeNode, key []byte) error {
	idx := 0
	for idx < len(node.Keys) && bytes.Compare(node.Keys[idx], key) < 0 {
		idx++
	}

	// Key found in this node
	if idx < len(node.Keys) && bytes.Equal(node.Keys[idx], key) {
		if node.Leaf {
			// Remove from leaf
			copy(node.Keys[idx:], node.Keys[idx+1:])
			copy(node.Values[idx:], node.Values[idx+1:])
			node.Keys = node.Keys[:len(node.Keys)-1]
			node.Values = node.Values[:len(node.Values)-1]
		} else {
			return t.deleteInternalNode(node, idx)
		}
	} else if node.Leaf {
		return ErrKeyNotFound
	} else {
		// Descend to child
		child := node.Children[idx]
		if len(child.Keys) == t.Order-1 {
			t.fillChild(node, idx)
			if len(node.Keys) > idx {
				idx++
			}
		}
		return t.delete(node.Children[idx], key)
	}

	return nil
}

// deleteInternalNode deletes from an internal node.
func (t *BTree) deleteInternalNode(node *BTreeNode, index int) error {
	key := node.Keys[index]
	child := node.Children[index]

	// Case 1: Child has enough keys
	if len(child.Keys) >= t.Order {
		predKey := t.getPredecessor(child)
		t.delete(child, predKey)
		copy(node.Keys[index], predKey)
	} else if index+1 < len(node.Children) && len(node.Children[index+1].Keys) >= t.Order {
		succKey := t.getSuccessor(node.Children[index+1])
		t.delete(node.Children[index+1], succKey)
		copy(node.Keys[index], succKey)
	} else {
		// Merge children
		t.mergeNodes(node, index)
		return t.delete(child, key)
	}

	return nil
}

// getPredecessor returns the predecessor key.
func (t *BTree) getPredecessor(node *BTreeNode) []byte {
	for !node.Leaf {
		node = node.Children[len(node.Children)-1]
	}
	if len(node.Keys) > 0 {
		return node.Keys[len(node.Keys)-1]
	}
	return nil
}

// getSuccessor returns the successor key.
func (t *BTree) getSuccessor(node *BTreeNode) []byte {
	for !node.Leaf {
		node = node.Children[0]
	}
	if len(node.Keys) > 0 {
		return node.Keys[0]
	}
	return nil
}

// fillChild ensures a child has at least t.Order-1 keys.
func (t *BTree) fillChild(parent *BTreeNode, index int) {
	if index > 0 && len(parent.Children[index-1].Keys) >= t.Order {
		t.borrowFromPrev(parent, index)
	} else if index < len(parent.Keys) && len(parent.Children[index+1].Keys) >= t.Order {
		t.borrowFromNext(parent, index)
	} else if index < len(parent.Keys) {
		t.mergeNodes(parent, index)
	} else {
		t.mergeNodes(parent, index-1)
	}
}

// borrowFromPrev borrows a key from the previous sibling.
func (t *BTree) borrowFromPrev(parent *BTreeNode, index int) {
	child := parent.Children[index]
	sibling := parent.Children[index-1]

	if child.Leaf {
		// Move key from parent to child
		child.Keys = append([][]byte{parent.Keys[index-1]}, child.Keys...)
		child.Values = append([][]byte{parent.Values[index-1]}, child.Values...)
		// Move key from sibling to parent
		parent.Keys[index-1] = sibling.Keys[len(sibling.Keys)-1]
		parent.Values[index-1] = sibling.Values[len(sibling.Values)-1]
		sibling.Keys = sibling.Keys[:len(sibling.Keys)-1]
		sibling.Values = sibling.Values[:len(sibling.Values)-1]
	} else {
		// Move key from parent to child
		child.Keys = append([][]byte{parent.Keys[index-1]}, child.Keys...)
		child.Children = append(child.Children, sibling.Children[len(sibling.Children)-1])
		// Move key from sibling to parent
		parent.Keys[index-1] = sibling.Keys[len(sibling.Keys)-1]
		sibling.Keys = sibling.Keys[:len(sibling.Keys)-1]
	}
}

// borrowFromNext borrows a key from the next sibling.
func (t *BTree) borrowFromNext(parent *BTreeNode, index int) {
	child := parent.Children[index]
	sibling := parent.Children[index+1]

	if child.Leaf {
		// Move key from parent to child
		child.Keys = append(child.Keys, parent.Keys[index])
		child.Values = append(child.Values, parent.Values[index])
		// Move key from sibling to parent
		parent.Keys[index] = sibling.Keys[0]
		parent.Values[index] = sibling.Values[0]
		copy(sibling.Keys[0:], sibling.Keys[1:])
		copy(sibling.Values[0:], sibling.Values[1:])
		sibling.Keys = sibling.Keys[:len(sibling.Keys)-1]
		sibling.Values = sibling.Values[:len(sibling.Values)-1]
	} else {
		// Move key from parent to child
		child.Keys = append(child.Keys, parent.Keys[index])
		child.Children = append(child.Children, sibling.Children[0])
		// Move key from sibling to parent
		parent.Keys[index] = sibling.Keys[0]
		copy(sibling.Keys[0:], sibling.Keys[1:])
		sibling.Keys = sibling.Keys[:len(sibling.Keys)-1]
		copy(sibling.Children[0:], sibling.Children[1:])
		sibling.Children = sibling.Children[:len(sibling.Children)-1]
	}
}

// mergeNodes merges two siblings.
func (t *BTree) mergeNodes(parent *BTreeNode, index int) {
	child := parent.Children[index]
	sibling := parent.Children[index+1]

	// Pull down parent's key
	child.Keys = append(child.Keys, parent.Keys[index])
	child.Values = append(child.Values, parent.Values[index])

	// Copy sibling's keys and values
	child.Keys = append(child.Keys, sibling.Keys...)
	child.Values = append(child.Values, sibling.Values...)

	// Copy sibling's children if internal
	if !sibling.Leaf {
		child.Children = append(child.Children, sibling.Children...)
	}

	// Remove parent's key and sibling
	copy(parent.Keys[index:], parent.Keys[index+1:])
	copy(parent.Values[index:], parent.Values[index+1:])
	parent.Keys = parent.Keys[:len(parent.Keys)-1]
	parent.Values = parent.Values[:len(parent.Values)-1]
	copy(parent.Children[index+1:], parent.Children[index+2:])
	parent.Children = parent.Children[:len(parent.Children)-1]

	// If root became empty
	if len(parent.Keys) == 0 && parent != t.Root {
		t.Root = child
	}
}

// RangeQuery returns all key-value pairs where start <= key < end.
func (t *BTree) RangeQuery(start, end []byte) ([]*IndexEntry, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, ErrIndexClosed
	}

	var entries []*IndexEntry
	t.rangeQuery(t.Root, start, end, &entries)
	return entries, nil
}

// rangeQuery performs a range query recursively.
func (t *BTree) rangeQuery(node *BTreeNode, start, end []byte, entries *[]*IndexEntry) {
	if node == nil {
		return
	}

	for i, key := range node.Keys {
		if (start == nil || bytes.Compare(key, start) >= 0) &&
			(end == nil || bytes.Compare(key, end) < 0) {
			if node.Leaf {
				*entries = append(*entries, NewIndexEntry(key, node.Values[i]))
			}
		}
	}

	if !node.Leaf {
		for i, child := range node.Children {
			if i < len(node.Keys) {
				if end == nil || bytes.Compare(node.Keys[i], end) < 0 {
					t.rangeQuery(child, start, end, entries)
				}
			} else {
				t.rangeQuery(child, start, end, entries)
			}
		}
	}
}

// Min returns the minimum key in the B-tree.
func (t *BTree) Min() ([]byte, []byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, nil, ErrIndexClosed
	}

	node := t.Root
	for !node.Leaf {
		node = node.Children[0]
	}
	if len(node.Keys) == 0 {
		return nil, nil, ErrKeyNotFound
	}
	return node.Keys[0], node.Values[0], nil
}

// Max returns the maximum key in the B-tree.
func (t *BTree) Max() ([]byte, []byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, nil, ErrIndexClosed
	}

	node := t.Root
	for !node.Leaf {
		node = node.Children[len(node.Children)-1]
	}
	if len(node.Keys) == 0 {
		return nil, nil, ErrKeyNotFound
	}
	return node.Keys[len(node.Keys)-1], node.Values[len(node.Keys)-1], nil
}

// Close closes the B-tree.
func (t *BTree) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.closed = true
	t.Root = nil
	t.Count = 0
	return nil
}

// Index is a higher-level index interface using B-tree.
type Index struct {
	bt      *BTree
	name    string
	unique  bool
	table   string
	columns []string
	mu      sync.RWMutex
	closed  bool
}

// NewIndex creates a new index.
func NewIndex(name, table string, columns []string, unique bool) (*Index, error) {
	bt, err := NewBTree(DefaultOrder)
	if err != nil {
		return nil, err
	}
	return &Index{
		bt:      bt,
		name:    name,
		table:   table,
		columns: columns,
		unique:  unique,
	}, nil
}

// Name returns the index name.
func (i *Index) Name() string {
	return i.name
}

// Table returns the table name.
func (i *Index) Table() string {
	return i.table
}

// Columns returns the indexed columns.
func (i *Index) Columns() []string {
	cols := make([]string, len(i.columns))
	copy(cols, i.columns)
	return cols
}

// Unique returns whether the index is unique.
func (i *Index) Unique() bool {
	return i.unique
}

// Insert inserts a key-value pair into the index.
func (i *Index) Insert(key, value []byte) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return ErrIndexClosed
	}

	return i.bt.Insert(key, value)
}

// Search searches for a key in the index.
func (i *Index) Search(key []byte) ([]byte, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.closed {
		return nil, ErrIndexClosed
	}

	return i.bt.Search(key)
}

// Delete deletes a key from the index.
func (i *Index) Delete(key []byte) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return ErrIndexClosed
	}

	return i.bt.Delete(key)
}

// RangeQuery performs a range query on the index.
func (i *Index) RangeQuery(start, end []byte) ([]*IndexEntry, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.closed {
		return nil, ErrIndexClosed
	}

	return i.bt.RangeQuery(start, end)
}

// Close closes the index.
func (i *Index) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.closed = true
	return i.bt.Close()
}

// IndexManager manages multiple indexes on a table.
type IndexManager struct {
	indexes map[string]*Index
	table   string
	mu      sync.RWMutex
}

// NewIndexManager creates a new index manager.
func NewIndexManager(table string) *IndexManager {
	return &IndexManager{
		indexes: make(map[string]*Index),
		table:   table,
	}
}

// CreateIndex creates a new index.
func (m *IndexManager) CreateIndex(name string, columns []string, unique bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.indexes[name]; exists {
		return fmt.Errorf("index %s already exists", name)
	}

	idx, err := NewIndex(name, m.table, columns, unique)
	if err != nil {
		return err
	}

	m.indexes[name] = idx
	return nil
}

// GetIndex returns an index by name.
func (m *IndexManager) GetIndex(name string) (*Index, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idx, ok := m.indexes[name]
	return idx, ok
}

// DropIndex drops an index by name.
func (m *IndexManager) DropIndex(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, ok := m.indexes[name]
	if !ok {
		return fmt.Errorf("index %s not found", name)
	}

	if err := idx.Close(); err != nil {
		return err
	}

	delete(m.indexes, name)
	return nil
}

// Close closes all indexes.
func (m *IndexManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, idx := range m.indexes {
		if err := idx.Close(); err != nil {
			return err
		}
	}
	m.indexes = make(map[string]*Index)
	return nil
}
