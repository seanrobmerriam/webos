# Database Engine

A lightweight SQL database engine implemented in Go with ACID transaction support.

## Architecture

### Core Components

```
pkg/database/
├── database.go         # Database core and catalog management
├── table.go            # Table management and row operations
├── schema.go           # Schema definition and column types
├── index.go            # B-tree index implementation
├── query/
│   ├── parser.go       # SQL parser (tokenizer, expression parser)
│   ├── planner.go      # Query planner
│   └── executor.go     # Query executor
├── txn/
│   ├── manager.go      # Transaction manager (ACID support)
│   └── lock.go         # Lock manager for isolation
└── recovery/
    ├── log.go          # Write-Ahead Log (WAL)
    └── checkpoint.go   # Checkpoint management
```

### Data Flow

```
SQL Query
    ↓
Parser (tokenization → AST)
    ↓
Planner (query plan generation)
    ↓
Executor (plan execution)
    ↓
Table/Index (data access)
    ↓
Transaction Manager (ACID)
    ↓
Recovery (WAL logging)
```

### Key Abstractions

- **Database**: Manages tables and catalog
- **Table**: Stores rows with schema validation
- **Schema**: Defines column types and constraints
- **Index**: B-tree structure for fast lookups
- **Transaction**: ACID-compliant transaction wrapper
- **WAL**: Write-Ahead Log for crash recovery

## Features

### SQL Support

#### Data Definition Language (DDL)
- `CREATE TABLE` - Create new tables with schema
- `ALTER TABLE` - Add columns to existing tables
- `DROP TABLE` - Remove tables from database

#### Data Manipulation Language (DML)
- `SELECT` - Query data with filtering and sorting
- `INSERT` - Insert new rows
- `UPDATE` - Update existing rows
- `DELETE` - Delete rows

#### Advanced Query Features
- **JOINs**: INNER, LEFT, RIGHT, CROSS joins
- **Aggregation**: GROUP BY, HAVING, COUNT(*)
- **Filtering**: WHERE clauses with AND/OR/NOT
- **Sorting**: ORDER BY (ASC/DESC)
- **Limits**: LIMIT and OFFSET for pagination
- **DISTINCT**: Remove duplicate results
- **IS NULL**: NULL value checking

#### Data Types
- `INTEGER` - 64-bit signed integer
- `TEXT` - Variable-length text
- `FLOAT` - 64-bit floating point
- `BOOLEAN` - true/false values

#### Constraints
- `PRIMARY KEY` - Primary key constraint
- `NOT NULL` - Non-null constraint
- `UNIQUE` - Unique value constraint
- `DEFAULT` - Default value specification

### Transaction Support (ACID)

- **Atomicity**: All changes in a transaction succeed or fail together
- **Consistency**: Database transitions between valid states
- **Isolation**: Concurrent transactions don't interfere
- **Durability**: Committed changes survive crashes

#### Transaction Lifecycle
1. `Begin()` - Start transaction
2. Execute DML operations
3. `Commit()` - Make changes permanent
4. `Rollback()` - Discard changes

### Recovery System

- **Write-Ahead Logging (WAL)**: All changes logged before application
- **Crash Recovery**: Replay logs to restore consistency
- **Checkpoint Support**: Periodic state snapshots

### B-tree Indexes

- Efficient O(log n) lookups
- Automatic index maintenance on inserts/updates/deletes
- Range query support

## Usage

### Running the Demo

```bash
# Build the demo
go build -o database-demo ./cmd/database-demo/

# Run the demo
./database-demo
```

### Programmatic Usage

```go
import "webos/pkg/database"

// Create database
db, err := database.Open(":memory:")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Create table
err = db.Exec(`
    CREATE TABLE users (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL,
        email TEXT UNIQUE,
        age INTEGER DEFAULT 0
    )
`)
if err != nil {
    log.Fatal(err)
}

// Insert data
_, err = db.Exec("INSERT INTO users (id, name, email, age) VALUES (1, 'Alice', 'alice@example.com', 30)")
if err != nil {
    log.Fatal(err)
}

// Query data
result, err := db.Query("SELECT * FROM users WHERE age > ?", 25)
if err != nil {
    log.Fatal(err)
}
defer result.Close()

for result.Next() {
    var id int
    var name, email string
    var age int
    result.Scan(&id, &name, &email, &age)
    fmt.Printf("User: %s (%s)\n", name, email)
}
```

### Transaction Example

```go
txn, err := db.Begin()
if err != nil {
    log.Fatal(err)
}

// Execute operations within transaction
_, err = txn.Exec("UPDATE users SET age = age + 1 WHERE id = 1")
if err != nil {
    txn.Rollback()
    log.Fatal(err)
}

// Commit transaction
if err = txn.Commit(); err != nil {
    log.Fatal(err)
}
```

## Testing

```bash
# Run all database tests
go test ./pkg/database/... -v

# Run with race detection
go test ./pkg/database/... -race

# Run with coverage
go test ./pkg/database/... -cover
```

## Performance Characteristics

| Operation | Complexity |
|-----------|------------|
| Table Scan | O(n) |
| Indexed Lookup | O(log n) |
| Insert (indexed) | O(log n) |
| Update (indexed) | O(log n) |
| Delete (indexed) | O(log n) |
| Join (nested loop) | O(n*m) |

## Limitations

- No automatic index creation (must be manually created)
- Limited to single-writer transactions
- No query optimization beyond basic planning
- In-memory only storage (no persistent mode in current implementation)
- No support for: VIEWs, stored procedures, triggers, constraints (FOREIGN KEY, CHECK)

## Future Enhancements

- Persistent storage with block manager
- Query optimizer with cost-based planning
- Automatic index creation
- Foreign key constraints
- VIEW support
- Subquery optimization
- Prepared statements
