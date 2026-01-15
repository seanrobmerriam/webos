// Package query provides SQL parsing, planning, and execution.
package query

import (
	"errors"
	"fmt"
)

// Planner errors.
var (
	ErrTableNotFound    = errors.New("table not found")
	ErrColumnNotFound   = errors.New("column not found")
	ErrAmbiguousColumn  = errors.New("ambiguous column reference")
	ErrInvalidOperation = errors.New("invalid operation")
)

// PlanNode types.
type PlanNodeType int

const (
	PlanScan PlanNodeType = iota
	PlanFilter
	PlanProject
	PlanJoin
	PlanSort
	PlanLimit
	PlanAggregate
	PlanInsert
	PlanUpdate
	PlanDelete
)

// PlanNode represents a node in the query plan.
type PlanNode struct {
	Type       PlanNodeType
	Children   []*PlanNode
	Properties map[string]interface{}
	OutputCols []string
}

// QueryPlan represents an execution plan for a query.
type QueryPlan struct {
	PlanNode
	Root   *PlanNode
	Params map[string]interface{}
}

// Planner creates query execution plans from SQL statements.
type Planner struct {
	schemas map[string]interface{}
}

// NewPlanner creates a new query planner.
func NewPlanner() *Planner {
	return &Planner{
		schemas: make(map[string]interface{}),
	}
}

// SetSchema sets the schema for a table.
func (p *Planner) SetSchema(tableName string, schema interface{}) {
	p.schemas[tableName] = schema
}

// Plan creates an execution plan for a SQL statement.
func (p *Planner) Plan(stmt *Statement) (*QueryPlan, error) {
	switch stmt.Type {
	case StmtSelect:
		return p.planSelect(stmt.Select)
	case StmtInsert:
		return p.planInsert(stmt.Insert)
	case StmtUpdate:
		return p.planUpdate(stmt.Update)
	case StmtDelete:
		return p.planDelete(stmt.Delete)
	case StmtCreateTable:
		return p.planCreateTable(stmt.CreateTable)
	case StmtDropTable:
		return p.planDropTable(stmt.DropTable)
	default:
		return nil, fmt.Errorf("%w: unsupported statement type", ErrInvalidOperation)
	}
}

// planSelect creates an execution plan for a SELECT statement.
func (p *Planner) planSelect(stmt *SelectStatement) (*QueryPlan, error) {
	if _, ok := p.schemas[stmt.Table]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrTableNotFound, stmt.Table)
	}

	plan := &QueryPlan{
		Params: make(map[string]interface{}),
	}

	root := &PlanNode{
		Type:       PlanScan,
		Properties: map[string]interface{}{"table": stmt.Table},
	}

	for _, join := range stmt.Joins {
		joinNode := &PlanNode{
			Type: PlanJoin,
			Properties: map[string]interface{}{
				"type":      join.Type,
				"table":     join.Table,
				"alias":     join.Alias,
				"condition": join.Condition,
			},
			Children: []*PlanNode{root},
		}
		root = joinNode
	}

	if stmt.Where.Type != 0 {
		filterNode := &PlanNode{
			Type:       PlanFilter,
			Properties: map[string]interface{}{"condition": stmt.Where},
			Children:   []*PlanNode{root},
		}
		root = filterNode
	}

	if len(stmt.GroupBy) > 0 {
		aggNode := &PlanNode{
			Type: PlanAggregate,
			Properties: map[string]interface{}{
				"groupBy": stmt.GroupBy,
				"having":  stmt.Having,
			},
			Children: []*PlanNode{root},
		}
		root = aggNode
	}

	if len(stmt.OrderBy) > 0 {
		sortNode := &PlanNode{
			Type:       PlanSort,
			Properties: map[string]interface{}{"orderBy": stmt.OrderBy},
			Children:   []*PlanNode{root},
		}
		root = sortNode
	}

	if stmt.Limit > 0 {
		limitNode := &PlanNode{
			Type: PlanLimit,
			Properties: map[string]interface{}{
				"limit":  stmt.Limit,
				"offset": stmt.Offset,
			},
			Children: []*PlanNode{root},
		}
		root = limitNode
	}

	projectNode := &PlanNode{
		Type:       PlanProject,
		Properties: map[string]interface{}{"columns": stmt.Columns},
		Children:   []*PlanNode{root},
	}

	plan.Root = projectNode

	for _, col := range stmt.Columns {
		if col.Type == ExprColumn {
			if val, ok := col.Value.(string); ok {
				plan.OutputCols = append(plan.OutputCols, val)
			}
		}
	}

	return plan, nil
}

// planInsert creates an execution plan for an INSERT statement.
func (p *Planner) planInsert(stmt *InsertStatement) (*QueryPlan, error) {
	if _, ok := p.schemas[stmt.Table]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrTableNotFound, stmt.Table)
	}

	plan := &QueryPlan{
		Root: &PlanNode{
			Type: PlanInsert,
			Properties: map[string]interface{}{
				"table":   stmt.Table,
				"columns": stmt.Columns,
				"values":  stmt.Values,
			},
		},
		Params: make(map[string]interface{}),
	}

	return plan, nil
}

// planUpdate creates an execution plan for an UPDATE statement.
func (p *Planner) planUpdate(stmt *UpdateStatement) (*QueryPlan, error) {
	if _, ok := p.schemas[stmt.Table]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrTableNotFound, stmt.Table)
	}

	plan := &QueryPlan{
		Root: &PlanNode{
			Type: PlanUpdate,
			Properties: map[string]interface{}{
				"table":      stmt.Table,
				"setClauses": stmt.SetClauses,
				"where":      stmt.Where,
			},
		},
		Params: make(map[string]interface{}),
	}

	return plan, nil
}

// planDelete creates an execution plan for a DELETE statement.
func (p *Planner) planDelete(stmt *DeleteStatement) (*QueryPlan, error) {
	if _, ok := p.schemas[stmt.Table]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrTableNotFound, stmt.Table)
	}

	plan := &QueryPlan{
		Root: &PlanNode{
			Type: PlanDelete,
			Properties: map[string]interface{}{
				"table": stmt.Table,
				"where": stmt.Where,
			},
		},
		Params: make(map[string]interface{}),
	}

	return plan, nil
}

// planCreateTable creates an execution plan for a CREATE TABLE statement.
func (p *Planner) planCreateTable(stmt *CreateTableStatement) (*QueryPlan, error) {
	if _, ok := p.schemas[stmt.TableName]; ok {
		return nil, fmt.Errorf("table %s already exists", stmt.TableName)
	}

	plan := &QueryPlan{
		Root: &PlanNode{
			Type:       PlanScan,
			Properties: map[string]interface{}{"ddl": "CREATE TABLE", "tableName": stmt.TableName},
		},
		Params: make(map[string]interface{}),
	}

	return plan, nil
}

// planDropTable creates an execution plan for a DROP TABLE statement.
func (p *Planner) planDropTable(stmt *DropTableStatement) (*QueryPlan, error) {
	if _, ok := p.schemas[stmt.TableName]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrTableNotFound, stmt.TableName)
	}

	plan := &QueryPlan{
		Root: &PlanNode{
			Type:       PlanScan,
			Properties: map[string]interface{}{"ddl": "DROP TABLE", "tableName": stmt.TableName},
		},
		Params: make(map[string]interface{}),
	}

	return plan, nil
}

// String returns a string representation of the query plan.
func (p *QueryPlan) String() string {
	return printPlanNode(p.Root, 0)
}

// printPlanNode prints a plan node with indentation.
func printPlanNode(node *PlanNode, indent int) string {
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "  "
	}

	result := indentStr + planNodeTypeToString(node.Type)

	if len(node.Properties) > 0 {
		result += " ("
		first := true
		for k, v := range node.Properties {
			if !first {
				result += ", "
			}
			result += fmt.Sprintf("%s=%v", k, v)
			first = false
		}
		result += ")"
	}

	for _, child := range node.Children {
		result += "\n" + printPlanNode(child, indent+1)
	}

	return result
}

// planNodeTypeToString returns a string representation of a plan node type.
func planNodeTypeToString(t PlanNodeType) string {
	switch t {
	case PlanScan:
		return "SCAN"
	case PlanFilter:
		return "FILTER"
	case PlanProject:
		return "PROJECT"
	case PlanJoin:
		return "JOIN"
	case PlanSort:
		return "SORT"
	case PlanLimit:
		return "LIMIT"
	case PlanAggregate:
		return "AGGREGATE"
	case PlanInsert:
		return "INSERT"
	case PlanUpdate:
		return "UPDATE"
	case PlanDelete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}
