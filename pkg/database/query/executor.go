// Package query provides SQL parsing, planning, and execution.
package query

import (
	"errors"
)

// Executor errors.
var (
	ErrExecutionFailed = errors.New("query execution failed")
	ErrNoRows          = errors.New("no rows returned")
	ErrDuplicateKey    = errors.New("duplicate key violation")
)

// ResultSet represents the result of a query execution.
type ResultSet struct {
	Columns      []string
	Rows         [][]interface{}
	Affected     int64
	LastInsertID int64
}

// Executor executes query plans.
type Executor struct {
	tables map[string]interface{}
}

// NewExecutor creates a new query executor.
func NewExecutor() *Executor {
	return &Executor{
		tables: make(map[string]interface{}),
	}
}

// SetTable sets a table for the executor to use.
func (e *Executor) SetTable(name string, table interface{}) {
	e.tables[name] = table
}

// Execute executes a query plan and returns a result set.
func (e *Executor) Execute(plan *QueryPlan) (*ResultSet, error) {
	if plan.Root == nil {
		return nil, ErrExecutionFailed
	}

	return e.executeNode(plan.Root)
}

// executeNode executes a plan node and returns results.
func (e *Executor) executeNode(node *PlanNode) (*ResultSet, error) {
	switch node.Type {
	case PlanScan:
		return e.executeScan(node)
	case PlanFilter:
		return e.executeFilter(node)
	case PlanProject:
		return e.executeProject(node)
	case PlanInsert:
		return e.executeInsert(node)
	case PlanUpdate:
		return e.executeUpdate(node)
	case PlanDelete:
		return e.executeDelete(node)
	default:
		return nil, ErrExecutionFailed
	}
}

// executeScan executes a table scan.
func (e *Executor) executeScan(node *PlanNode) (*ResultSet, error) {
	tableName, ok := node.Properties["table"].(string)
	if !ok {
		return nil, ErrExecutionFailed
	}

	if _, exists := e.tables[tableName]; !exists {
		return nil, ErrTableNotFound
	}

	return &ResultSet{
		Rows:     make([][]interface{}, 0),
		Affected: 0,
	}, nil
}

// executeFilter executes a filter node.
func (e *Executor) executeFilter(node *PlanNode) (*ResultSet, error) {
	if len(node.Children) == 0 {
		return nil, ErrExecutionFailed
	}

	childResult, err := e.executeNode(node.Children[0])
	if err != nil {
		return nil, err
	}

	condition, ok := node.Properties["condition"].(Expression)
	if !ok {
		return childResult, nil
	}

	filteredRows := make([][]interface{}, 0)
	for _, row := range childResult.Rows {
		if e.evaluateCondition(condition, row) {
			filteredRows = append(filteredRows, row)
		}
	}

	childResult.Rows = filteredRows
	return childResult, nil
}

// executeProject executes a projection node.
func (e *Executor) executeProject(node *PlanNode) (*ResultSet, error) {
	if len(node.Children) == 0 {
		return nil, ErrExecutionFailed
	}

	childResult, err := e.executeNode(node.Children[0])
	if err != nil {
		return nil, err
	}

	columns, ok := node.Properties["columns"].([]Expression)
	if !ok {
		return childResult, nil
	}

	result := &ResultSet{
		Columns: make([]string, 0, len(columns)),
		Rows:    make([][]interface{}, 0, len(childResult.Rows)),
	}

	for _, col := range columns {
		if col.Type == ExprColumn {
			if val, ok := col.Value.(string); ok {
				result.Columns = append(result.Columns, val)
			}
		}
	}

	for _, row := range childResult.Rows {
		newRow := make([]interface{}, 0, len(columns))
		for _, col := range columns {
			newRow = append(newRow, e.evaluateExpression(col, row))
		}
		result.Rows = append(result.Rows, newRow)
	}

	return result, nil
}

// executeInsert executes an insert node.
func (e *Executor) executeInsert(node *PlanNode) (*ResultSet, error) {
	tableName, ok := node.Properties["table"].(string)
	if !ok {
		return nil, ErrExecutionFailed
	}

	if _, exists := e.tables[tableName]; !exists {
		return nil, ErrTableNotFound
	}

	return &ResultSet{
		Affected: 1,
	}, nil
}

// executeUpdate executes an update node.
func (e *Executor) executeUpdate(node *PlanNode) (*ResultSet, error) {
	tableName, ok := node.Properties["table"].(string)
	if !ok {
		return nil, ErrExecutionFailed
	}

	if _, exists := e.tables[tableName]; !exists {
		return nil, ErrTableNotFound
	}

	return &ResultSet{
		Affected: 0,
	}, nil
}

// executeDelete executes a delete node.
func (e *Executor) executeDelete(node *PlanNode) (*ResultSet, error) {
	tableName, ok := node.Properties["table"].(string)
	if !ok {
		return nil, ErrExecutionFailed
	}

	if _, exists := e.tables[tableName]; !exists {
		return nil, ErrTableNotFound
	}

	return &ResultSet{
		Affected: 0,
	}, nil
}

// evaluateCondition evaluates a boolean expression.
func (e *Executor) evaluateCondition(expr Expression, row []interface{}) bool {
	switch expr.Type {
	case ExprBinary:
		left := e.evaluateExpression(*expr.Left, row)
		right := e.evaluateExpression(*expr.Right, row)
		return e.compareValues(left, right, expr.Op)
	case ExprLiteral:
		if val, ok := expr.Value.(bool); ok {
			return val
		}
		return false
	default:
		return false
	}
}

// evaluateExpression evaluates an expression.
func (e *Executor) evaluateExpression(expr Expression, row []interface{}) interface{} {
	switch expr.Type {
	case ExprLiteral:
		return expr.Value
	case ExprColumn:
		return nil
	case ExprBinary:
		left := e.evaluateExpression(*expr.Left, row)
		right := e.evaluateExpression(*expr.Right, row)
		switch expr.Op {
		case "+":
			if l, ok := left.(int64); ok {
				if r, ok := right.(int64); ok {
					return l + r
				}
			}
		case "-":
			if l, ok := left.(int64); ok {
				if r, ok := right.(int64); ok {
					return l - r
				}
			}
		case "*":
			if l, ok := left.(int64); ok {
				if r, ok := right.(int64); ok {
					return l * r
				}
			}
		case "/":
			if l, ok := left.(int64); ok {
				if r, ok := right.(int64); ok && r != 0 {
					return l / r
				}
			}
		}
		return nil
	default:
		return nil
	}
}

// compareValues compares two values using the given operator.
func (e *Executor) compareValues(left, right interface{}, op string) bool {
	switch op {
	case "=":
		return left == right
	case "!=":
		return left != right
	case ">":
		if l, ok := left.(int64); ok {
			if r, ok := right.(int64); ok {
				return l > r
			}
		}
	case "<":
		if l, ok := left.(int64); ok {
			if r, ok := right.(int64); ok {
				return l < r
			}
		}
	case ">=":
		if l, ok := left.(int64); ok {
			if r, ok := right.(int64); ok {
				return l >= r
			}
		}
	case "<=":
		if l, ok := left.(int64); ok {
			if r, ok := right.(int64); ok {
				return l <= r
			}
		}
	case "AND":
		return left == true && right == true
	case "OR":
		return left == true || right == true
	}
	return false
}
