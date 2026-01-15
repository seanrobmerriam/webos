// Package txn provides transaction management with ACID support.
package txn

import (
	"errors"
	"sync"
	"time"
)

// Transaction states.
const (
	// StateActive indicates the transaction is active.
	StateActive uint8 = iota
	// StateCommitted indicates the transaction has been committed.
	StateCommitted
	// StateRolledBack indicates the transaction has been rolled back.
	StateRolledBack
)

// Transaction errors.
var (
	ErrInvalidTransaction    = errors.New("invalid transaction")
	ErrTransactionActive     = errors.New("transaction is already active")
	ErrTransactionNotActive  = errors.New("transaction is not active")
	ErrTransactionCommitted  = errors.New("transaction already committed")
	ErrTransactionRolledBack = errors.New("transaction already rolled back")
	ErrTooManyTransactions   = errors.New("too many concurrent transactions")
)

// Isolation levels.
type IsolationLevel uint8

const (
	// IsolationReadCommitted provides read committed isolation.
	IsolationReadCommitted IsolationLevel = iota
	// IsolationRepeatableRead provides repeatable read isolation.
	IsolationRepeatableRead
	// IsolationSerializable provides serializable isolation.
	IsolationSerializable
)

// String returns a string representation of the isolation level.
func (l IsolationLevel) String() string {
	switch l {
	case IsolationReadCommitted:
		return "READ COMMITTED"
	case IsolationRepeatableRead:
		return "REPEATABLE READ"
	case IsolationSerializable:
		return "SERIALIZABLE"
	default:
		return "UNKNOWN"
	}
}

// Transaction represents a database transaction.
type Transaction struct {
	ID             uint64            // Transaction ID
	State          uint8             // Transaction state
	Isolation      IsolationLevel    // Isolation level
	StartTime      time.Time         // Transaction start time
	Snapshot       []byte            // Snapshot data (for MVCC)
	modifiedRows   map[uint64][]byte // Modified row before images
	modifiedTables map[string]bool   // Modified tables
	mu             sync.Mutex        // Transaction mutex
}

// TransactionManager manages database transactions.
type TransactionManager struct {
	mu              sync.RWMutex            // Global mutex
	transactions    map[uint64]*Transaction // Active transactions
	completedStates map[uint64]uint8        // States of completed transactions
	nextTxID        uint64                  // Next transaction ID
	maxActiveTxns   int                     // Maximum concurrent transactions
	isolationLevel  IsolationLevel          // Default isolation level
}

// NewTransactionManager creates a new transaction manager.
func NewTransactionManager(maxActiveTxns int, isolationLevel IsolationLevel) *TransactionManager {
	return &TransactionManager{
		transactions:    make(map[uint64]*Transaction),
		completedStates: make(map[uint64]uint8),
		maxActiveTxns:   maxActiveTxns,
		isolationLevel:  isolationLevel,
		nextTxID:        1,
	}
}

// Begin starts a new transaction.
func (m *TransactionManager) Begin() (*Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we've reached the max concurrent transactions
	if len(m.transactions) >= m.maxActiveTxns {
		return nil, ErrTooManyTransactions
	}

	txn := &Transaction{
		ID:             m.nextTxID,
		State:          StateActive,
		Isolation:      m.isolationLevel,
		StartTime:      time.Now(),
		modifiedRows:   make(map[uint64][]byte),
		modifiedTables: make(map[string]bool),
	}

	m.transactions[txn.ID] = txn
	m.nextTxID++

	return txn, nil
}

// Get returns a transaction by ID.
func (m *TransactionManager) Get(txID uint64) (*Transaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txn, ok := m.transactions[txID]
	return txn, ok
}

// Commit commits a transaction.
func (m *TransactionManager) Commit(txID uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	txn, ok := m.transactions[txID]
	if !ok {
		// Check if transaction was already completed
		if state, ok := m.completedStates[txID]; ok {
			if state == StateCommitted {
				return ErrTransactionCommitted
			}
			return ErrTransactionRolledBack
		}
		return ErrInvalidTransaction
	}

	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != StateActive {
		if txn.State == StateCommitted {
			return ErrTransactionCommitted
		}
		return ErrTransactionRolledBack
	}

	txn.State = StateCommitted
	delete(m.transactions, txID)
	m.completedStates[txID] = StateCommitted

	return nil
}

// Rollback rolls back a transaction.
func (m *TransactionManager) Rollback(txID uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	txn, ok := m.transactions[txID]
	if !ok {
		// Check if transaction was already completed
		if state, ok := m.completedStates[txID]; ok {
			if state == StateRolledBack {
				return ErrTransactionRolledBack
			}
			return ErrTransactionCommitted
		}
		return ErrInvalidTransaction
	}

	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != StateActive {
		if txn.State == StateRolledBack {
			return ErrTransactionRolledBack
		}
		return ErrTransactionCommitted
	}

	txn.State = StateRolledBack
	delete(m.transactions, txID)
	m.completedStates[txID] = StateRolledBack

	return nil
}

// ActiveTransactions returns the count of active transactions.
func (m *TransactionManager) ActiveTransactions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.transactions)
}

// GetIsolationLevel returns the default isolation level.
func (m *TransactionManager) GetIsolationLevel() IsolationLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.isolationLevel
}

// SetIsolationLevel sets the default isolation level.
func (m *TransactionManager) SetIsolationLevel(level IsolationLevel) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.isolationLevel = level
}

// RecordBeforeImage records the before image of a modified row.
func (t *Transaction) RecordBeforeImage(rowID uint64, beforeImage []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.modifiedRows[rowID]; !exists {
		t.modifiedRows[rowID] = beforeImage
	}
}

// GetBeforeImage returns the before image of a modified row.
func (t *Transaction) GetBeforeImage(rowID uint64) []byte {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.modifiedRows[rowID]
}

// MarkTableModified marks a table as modified in this transaction.
func (t *Transaction) MarkTableModified(tableName string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.modifiedTables[tableName] = true
}

// IsTableModified checks if a table was modified in this transaction.
func (t *Transaction) IsTableModified(tableName string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.modifiedTables[tableName]
}

// GetModifiedTables returns the list of modified tables.
func (t *Transaction) GetModifiedTables() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	tables := make([]string, 0, len(t.modifiedTables))
	for table := range t.modifiedTables {
		tables = append(tables, table)
	}
	return tables
}

// IsActive checks if the transaction is active.
func (t *Transaction) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.State == StateActive
}

// IsCommitted checks if the transaction is committed.
func (t *Transaction) IsCommitted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.State == StateCommitted
}

// IsRolledBack checks if the transaction is rolled back.
func (t *Transaction) IsRolledBack() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.State == StateRolledBack
}
