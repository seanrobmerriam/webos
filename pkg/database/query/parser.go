// Package query provides SQL parsing, planning, and execution.
package query

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Parser errors.
var (
	ErrSyntaxError       = errors.New("SQL syntax error")
	ErrUnexpectedToken   = errors.New("unexpected token")
	ErrInvalidColumn     = errors.New("invalid column reference")
	ErrInvalidTable      = errors.New("invalid table reference")
	ErrInvalidValue      = errors.New("invalid value")
	ErrUnsupportedSyntax = errors.New("unsupported SQL syntax")
)

// Token types.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenString
	TokenNumber
	TokenSymbol
	TokenKeyword
)

// Token represents a SQL token.
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Keywords.
var keywords = map[string]TokenType{
	"SELECT":        TokenKeyword,
	"FROM":          TokenKeyword,
	"WHERE":         TokenKeyword,
	"INSERT":        TokenKeyword,
	"INTO":          TokenKeyword,
	"VALUES":        TokenKeyword,
	"UPDATE":        TokenKeyword,
	"SET":           TokenKeyword,
	"DELETE":        TokenKeyword,
	"CREATE":        TokenKeyword,
	"TABLE":         TokenKeyword,
	"DROP":          TokenKeyword,
	"ALTER":         TokenKeyword,
	"ADD":           TokenKeyword,
	"PRIMARY":       TokenKeyword,
	"KEY":           TokenKeyword,
	"NOT":           TokenKeyword,
	"NULL":          TokenKeyword,
	"UNIQUE":        TokenKeyword,
	"AUTOINCREMENT": TokenKeyword,
	"DEFAULT":       TokenKeyword,
	"AND":           TokenKeyword,
	"OR":            TokenKeyword,
	"IN":            TokenKeyword,
	"LIKE":          TokenKeyword,
	"ORDER":         TokenKeyword,
	"BY":            TokenKeyword,
	"ASC":           TokenKeyword,
	"DESC":          TokenKeyword,
	"LIMIT":         TokenKeyword,
	"OFFSET":        TokenKeyword,
	"GROUP":         TokenKeyword,
	"HAVING":        TokenKeyword,
	"JOIN":          TokenKeyword,
	"INNER":         TokenKeyword,
	"LEFT":          TokenKeyword,
	"RIGHT":         TokenKeyword,
	"OUTER":         TokenKeyword,
	"CROSS":         TokenKeyword,
	"AS":            TokenKeyword,
	"ON":            TokenKeyword,
	"IS":            TokenKeyword,
	"DISTINCT":      TokenKeyword,
	"TRUE":          TokenKeyword,
	"FALSE":         TokenKeyword,
}

// Symbols - sorted by length (longest first for proper matching)
var symbols = []string{
	">=", "<=", "<>", "!=",
	"*", ",", "(", ")", "=", ">", "<", "+", "-", ";", ".",
}

// Statement types.
type StatementType int

const (
	StmtUnknown StatementType = iota
	StmtSelect
	StmtInsert
	StmtUpdate
	StmtDelete
	StmtCreateTable
	StmtDropTable
	StmtAlterTable
)

// Statement represents a SQL statement.
type Statement struct {
	Type StatementType
	// For SELECT
	Select *SelectStatement
	// For INSERT
	Insert *InsertStatement
	// For UPDATE
	Update *UpdateStatement
	// For DELETE
	Delete *DeleteStatement
	// For CREATE TABLE
	CreateTable *CreateTableStatement
	// For DROP TABLE
	DropTable *DropTableStatement
	// For ALTER TABLE
	AlterTable *AlterTableStatement
}

// SelectStatement represents a SELECT query.
type SelectStatement struct {
	Columns  []Expression
	Table    string
	Joins    []JoinClause
	Where    Expression
	OrderBy  []OrderByClause
	GroupBy  []Expression
	Having   Expression
	Limit    int64
	Offset   int64
	Distinct bool
}

// InsertStatement represents an INSERT statement.
type InsertStatement struct {
	Table   string
	Columns []string
	Values  [][]Expression
}

// UpdateStatement represents an UPDATE statement.
type UpdateStatement struct {
	Table      string
	SetClauses []SetClause
	Where      Expression
}

// DeleteStatement represents a DELETE statement.
type DeleteStatement struct {
	Table string
	Where Expression
}

// CreateTableStatement represents a CREATE TABLE statement.
type CreateTableStatement struct {
	TableName  string
	Columns    []ColumnDefinition
	PrimaryKey []string
}

// DropTableStatement represents a DROP TABLE statement.
type DropTableStatement struct {
	TableName string
}

// AlterTableStatement represents an ALTER TABLE statement.
type AlterTableStatement struct {
	TableName string
	Action    AlterAction
}

// AlterAction represents an ALTER TABLE action.
type AlterAction interface {
	isAlterAction()
}

// AddColumnAction represents ADD COLUMN action.
type AddColumnAction struct {
	Column ColumnDefinition
}

func (a *AddColumnAction) isAlterAction() {}

// DropColumnAction represents DROP COLUMN action.
type DropColumnAction struct {
	ColumnName string
}

func (a *DropColumnAction) isAlterAction() {}

// SetClause represents a SET clause in UPDATE.
type SetClause struct {
	Column string
	Value  Expression
}

// JoinClause represents a JOIN clause.
type JoinClause struct {
	Type      string
	Table     string
	Alias     string
	Condition Expression
}

// OrderByClause represents an ORDER BY clause.
type OrderByClause struct {
	Column Expression
	Desc   bool
}

// ColumnDefinition represents a column definition.
type ColumnDefinition struct {
	Name       string
	Type       string
	NotNull    bool
	PrimaryKey bool
	Unique     bool
	AutoInc    bool
	Default    Expression
}

// Expression types.
type ExpressionType int

const (
	ExprLiteral ExpressionType = iota
	ExprColumn
	ExprBinary
	ExprUnary
	ExprFunction
	ExprSubquery
	ExprBetween
	ExprIn
	ExprLike
)

// Expression represents a SQL expression.
type Expression struct {
	Type  ExpressionType
	Value interface{}
	Left  *Expression
	Right *Expression
	Op    string
}

// Parser represents a SQL parser.
type Parser struct {
	input  string
	pos    int
	tokens []Token
	tokPos int
}

// NewParser creates a new SQL parser.
func NewParser(input string) *Parser {
	return &Parser{
		input:  input,
		pos:    0,
		tokPos: 0,
		tokens: nil,
	}
}

// Parse parses a SQL statement.
func (p *Parser) Parse() (*Statement, error) {
	p.tokenize()

	if len(p.tokens) == 0 {
		return nil, ErrSyntaxError
	}

	return p.parseStatement()
}

// isAlpha returns true if the byte is a letter or underscore.
func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

// isDigit returns true if the byte is a digit.
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// isAlnum returns true if the byte is alphanumeric.
func isAlnum(b byte) bool {
	return isAlpha(b) || isDigit(b)
}

// isSpace returns true if the byte is whitespace.
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// tokenize tokenizes the input string.
func (p *Parser) tokenize() {
	p.tokens = make([]Token, 0)
	p.pos = 0

	input := strings.TrimSpace(p.input)

	for p.pos < len(input) {
		ch := input[p.pos]

		// Skip whitespace
		if isSpace(ch) {
			p.pos++
			continue
		}

		// Check for symbols (sorted by length, longest first)
		remaining := input[p.pos:]
		matchedSymbol := ""
		for _, sym := range symbols {
			if strings.HasPrefix(remaining, sym) {
				matchedSymbol = sym
				break
			}
		}
		if matchedSymbol != "" {
			p.tokens = append(p.tokens, Token{
				Type:  TokenSymbol,
				Value: matchedSymbol,
				Pos:   p.pos,
			})
			p.pos += len(matchedSymbol)
			continue
		}

		// Check for string literals
		if ch == '"' || ch == '\'' {
			p.pos++
			start := p.pos
			for p.pos < len(input) && input[p.pos] != ch {
				// Handle escape sequences
				if input[p.pos] == '\\' && p.pos+1 < len(input) {
					p.pos += 2
					continue
				}
				p.pos++
			}
			value := ""
			if p.pos < len(input) {
				value = input[start:p.pos]
				p.pos++ // Skip closing quote
			}
			p.tokens = append(p.tokens, Token{
				Type:  TokenString,
				Value: value,
				Pos:   start,
			})
			continue
		}

		// Check for numbers
		if isDigit(ch) {
			start := p.pos
			for p.pos < len(input) && (isDigit(input[p.pos]) || input[p.pos] == '.') {
				p.pos++
			}
			p.tokens = append(p.tokens, Token{
				Type:  TokenNumber,
				Value: input[start:p.pos],
				Pos:   start,
			})
			continue
		}

		// Parse identifier or keyword
		start := p.pos
		for p.pos < len(input) && isAlnum(input[p.pos]) {
			p.pos++
		}
		value := input[start:p.pos]
		valueUpper := strings.ToUpper(value)

		if typ, ok := keywords[valueUpper]; ok {
			p.tokens = append(p.tokens, Token{
				Type:  typ,
				Value: valueUpper,
				Pos:   start,
			})
		} else {
			p.tokens = append(p.tokens, Token{
				Type:  TokenIdentifier,
				Value: value,
				Pos:   start,
			})
		}
	}

	// Add EOF token
	p.tokens = append(p.tokens, Token{
		Type:  TokenEOF,
		Value: "",
		Pos:   len(input),
	})
}

// peek returns the current token without consuming it.
func (p *Parser) peek() Token {
	if p.tokPos < len(p.tokens) {
		return p.tokens[p.tokPos]
	}
	return Token{Type: TokenEOF}
}

// next consumes and returns the current token.
func (p *Parser) next() Token {
	if p.tokPos < len(p.tokens) {
		tok := p.tokens[p.tokPos]
		p.tokPos++
		return tok
	}
	return Token{Type: TokenEOF}
}

// expect expects a specific token type and consumes it.
func (p *Parser) expect(typ TokenType) (Token, error) {
	tok := p.next()
	if tok.Type != typ {
		return tok, fmt.Errorf("%w: expected %v, got %v", ErrUnexpectedToken, typ, tok.Type)
	}
	return tok, nil
}

// parseStatement parses a SQL statement.
func (p *Parser) parseStatement() (*Statement, error) {
	tok := p.peek()

	switch tok.Type {
	case TokenKeyword:
		switch tok.Value {
		case "SELECT":
			return p.parseSelect()
		case "INSERT":
			return p.parseInsert()
		case "UPDATE":
			return p.parseUpdate()
		case "DELETE":
			return p.parseDelete()
		case "CREATE":
			return p.parseCreate()
		case "DROP":
			return p.parseDrop()
		case "ALTER":
			return p.parseAlter()
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrUnsupportedSyntax, tok.Value)
}

// parseSelect parses a SELECT statement.
func (p *Parser) parseSelect() (*Statement, error) {
	p.next() // Skip SELECT

	stmt := &Statement{
		Type:   StmtSelect,
		Select: &SelectStatement{},
	}

	// Parse DISTINCT
	if p.peek().Type == TokenKeyword && p.peek().Value == "DISTINCT" {
		p.next()
		stmt.Select.Distinct = true
	}

	// Parse columns
	stmt.Select.Columns = p.parseColumnList()

	// Parse FROM
	tok := p.peek()
	if tok.Type != TokenKeyword || tok.Value != "FROM" {
		return nil, fmt.Errorf("%w: expected FROM", ErrSyntaxError)
	}
	p.next() // Skip FROM

	// Parse table name
	tableTok := p.next()
	if tableTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected table name", ErrSyntaxError)
	}
	stmt.Select.Table = tableTok.Value

	// Parse JOINs
	for {
		tok := p.peek()
		if tok.Type == TokenKeyword {
			if tok.Value == "INNER" || tok.Value == "LEFT" || tok.Value == "RIGHT" || tok.Value == "CROSS" || tok.Value == "JOIN" {
				join, err := p.parseJoin()
				if err != nil {
					return nil, err
				}
				stmt.Select.Joins = append(stmt.Select.Joins, *join)
				continue
			}
		}
		break
	}

	// Parse WHERE
	if p.peek().Type == TokenKeyword && p.peek().Value == "WHERE" {
		p.next()
		where, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		stmt.Select.Where = *where
	}

	// Parse GROUP BY
	if p.peek().Type == TokenKeyword && p.peek().Value == "GROUP" {
		p.next() // Skip GROUP
		if tok := p.next(); tok.Type != TokenKeyword || tok.Value != "BY" {
			return nil, fmt.Errorf("%w: expected BY", ErrSyntaxError)
		}
		stmt.Select.GroupBy = p.parseGroupBy()
	}

	// Parse HAVING
	if p.peek().Type == TokenKeyword && p.peek().Value == "HAVING" {
		p.next()
		having, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		stmt.Select.Having = *having
	}

	// Parse ORDER BY
	if p.peek().Type == TokenKeyword && p.peek().Value == "ORDER" {
		p.next()
		if tok := p.next(); tok.Type != TokenKeyword || tok.Value != "BY" {
			return nil, fmt.Errorf("%w: expected BY", ErrSyntaxError)
		}
		stmt.Select.OrderBy = p.parseOrderBy()
	}

	// Parse LIMIT
	if p.peek().Type == TokenKeyword && p.peek().Value == "LIMIT" {
		p.next()
		limitTok := p.next()
		if limitTok.Type != TokenNumber {
			return nil, fmt.Errorf("%w: expected number", ErrSyntaxError)
		}
		limit, err := strconv.ParseInt(limitTok.Value, 10, 64)
		if err != nil {
			return nil, err
		}
		stmt.Select.Limit = limit
	}

	// Parse OFFSET
	if p.peek().Type == TokenKeyword && p.peek().Value == "OFFSET" {
		p.next()
		offsetTok := p.next()
		if offsetTok.Type != TokenNumber {
			return nil, fmt.Errorf("%w: expected number", ErrSyntaxError)
		}
		offset, err := strconv.ParseInt(offsetTok.Value, 10, 64)
		if err != nil {
			return nil, err
		}
		stmt.Select.Offset = offset
	}

	return stmt, nil
}

// parseInsert parses an INSERT statement.
func (p *Parser) parseInsert() (*Statement, error) {
	p.next() // Skip INSERT

	// Expect INTO
	if tok := p.next(); tok.Type != TokenKeyword || tok.Value != "INTO" {
		return nil, fmt.Errorf("%w: expected INTO", ErrSyntaxError)
	}

	// Parse table name
	tableTok := p.next()
	if tableTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected table name", ErrSyntaxError)
	}

	stmt := &Statement{
		Type: StmtInsert,
		Insert: &InsertStatement{
			Table: tableTok.Value,
		},
	}

	// Parse column list if present
	if p.peek().Type == TokenSymbol && p.peek().Value == "(" {
		p.next() // Skip (
		stmt.Insert.Columns = p.parseIdentifierList()
		if tok, err := p.expect(TokenSymbol); err != nil {
			return nil, err
		} else if tok.Value != ")" {
			return nil, fmt.Errorf("%w: expected )", ErrSyntaxError)
		}
	}

	// Expect VALUES
	if tok := p.next(); tok.Type != TokenKeyword || tok.Value != "VALUES" {
		return nil, fmt.Errorf("%w: expected VALUES", ErrSyntaxError)
	}

	// Parse value lists
	for {
		// Check for comma between value lists
		if p.peek().Type == TokenSymbol && p.peek().Value == "," {
			p.next() // Skip comma
		}

		// Check if we have a value list
		if p.peek().Type == TokenSymbol && p.peek().Value == "(" {
			values, err := p.parseValueList()
			if err != nil {
				return nil, err
			}
			stmt.Insert.Values = append(stmt.Insert.Values, values)
			continue
		}
		break
	}

	return stmt, nil
}

// parseUpdate parses an UPDATE statement.
func (p *Parser) parseUpdate() (*Statement, error) {
	p.next() // Skip UPDATE

	// Parse table name
	tableTok := p.next()
	if tableTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected table name", ErrSyntaxError)
	}

	stmt := &Statement{
		Type: StmtUpdate,
		Update: &UpdateStatement{
			Table: tableTok.Value,
		},
	}

	// Expect SET
	if tok := p.next(); tok.Type != TokenKeyword || tok.Value != "SET" {
		return nil, fmt.Errorf("%w: expected SET", ErrSyntaxError)
	}

	// Parse SET clauses
	for {
		setClause, err := p.parseSetClause()
		if err != nil {
			return nil, err
		}
		stmt.Update.SetClauses = append(stmt.Update.SetClauses, *setClause)

		// Check for more SET clauses or WHERE
		if p.peek().Type == TokenSymbol && p.peek().Value == "," {
			p.next()
			continue
		}
		break
	}

	// Parse WHERE
	if p.peek().Type == TokenKeyword && p.peek().Value == "WHERE" {
		p.next()
		where, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		stmt.Update.Where = *where
	}

	return stmt, nil
}

// parseDelete parses a DELETE statement.
func (p *Parser) parseDelete() (*Statement, error) {
	p.next() // Skip DELETE

	// Expect FROM
	if tok := p.next(); tok.Type != TokenKeyword || tok.Value != "FROM" {
		return nil, fmt.Errorf("%w: expected FROM", ErrSyntaxError)
	}

	// Parse table name
	tableTok := p.next()
	if tableTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected table name", ErrSyntaxError)
	}

	stmt := &Statement{
		Type: StmtDelete,
		Delete: &DeleteStatement{
			Table: tableTok.Value,
		},
	}

	// Parse WHERE
	if p.peek().Type == TokenKeyword && p.peek().Value == "WHERE" {
		p.next()
		where, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		stmt.Delete.Where = *where
	}

	return stmt, nil
}

// parseCreate parses a CREATE statement.
func (p *Parser) parseCreate() (*Statement, error) {
	p.next() // Skip CREATE

	tok := p.next()
	if tok.Type != TokenKeyword {
		return nil, fmt.Errorf("%w: expected TABLE", ErrSyntaxError)
	}

	if tok.Value != "TABLE" {
		return nil, fmt.Errorf("%w: only CREATE TABLE is supported", ErrUnsupportedSyntax)
	}

	// Parse table name
	tableTok := p.next()
	if tableTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected table name", ErrSyntaxError)
	}

	stmt := &Statement{
		Type: StmtCreateTable,
		CreateTable: &CreateTableStatement{
			TableName: tableTok.Value,
		},
	}

	// Expect (
	if p.peek().Type == TokenSymbol && p.peek().Value == "(" {
		p.next() // Skip (
	} else {
		return nil, fmt.Errorf("%w: expected (", ErrSyntaxError)
	}

	// Parse column definitions
	for p.peek().Type != TokenSymbol || p.peek().Value != ")" {
		col, err := p.parseColumnDefinition()
		if err != nil {
			return nil, err
		}
		stmt.CreateTable.Columns = append(stmt.CreateTable.Columns, *col)

		// Check for comma
		if p.peek().Type == TokenSymbol && p.peek().Value == "," {
			p.next()
		}
	}

	// Expect )
	if p.peek().Type == TokenSymbol && p.peek().Value == ")" {
		p.next() // Skip )
	}

	return stmt, nil
}

// parseDrop parses a DROP statement.
func (p *Parser) parseDrop() (*Statement, error) {
	p.next() // Skip DROP

	tok := p.next()
	if tok.Type != TokenKeyword {
		return nil, fmt.Errorf("%w: expected TABLE", ErrSyntaxError)
	}

	if tok.Value != "TABLE" {
		return nil, fmt.Errorf("%w: only DROP TABLE is supported", ErrUnsupportedSyntax)
	}

	// Parse table name
	tableTok := p.next()
	if tableTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected table name", ErrSyntaxError)
	}

	return &Statement{
		Type: StmtDropTable,
		DropTable: &DropTableStatement{
			TableName: tableTok.Value,
		},
	}, nil
}

// parseAlter parses an ALTER TABLE statement.
func (p *Parser) parseAlter() (*Statement, error) {
	p.next() // Skip ALTER

	// Expect TABLE
	tok := p.next()
	if tok.Type != TokenKeyword || tok.Value != "TABLE" {
		return nil, fmt.Errorf("%w: expected TABLE", ErrSyntaxError)
	}

	// Parse table name
	tableTok := p.next()
	if tableTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected table name", ErrSyntaxError)
	}

	stmt := &Statement{
		Type: StmtAlterTable,
		AlterTable: &AlterTableStatement{
			TableName: tableTok.Value,
		},
	}

	// Parse action
	actionTok := p.next()
	if actionTok.Type != TokenKeyword {
		return nil, fmt.Errorf("%w: expected ADD or DROP", ErrSyntaxError)
	}

	switch actionTok.Value {
	case "ADD":
		col, err := p.parseColumnDefinition()
		if err != nil {
			return nil, err
		}
		stmt.AlterTable.Action = &AddColumnAction{Column: *col}
	case "DROP":
		colTok := p.next()
		if colTok.Type != TokenIdentifier {
			return nil, fmt.Errorf("%w: expected column name", ErrSyntaxError)
		}
		stmt.AlterTable.Action = &DropColumnAction{ColumnName: colTok.Value}
	default:
		return nil, fmt.Errorf("%w: expected ADD or DROP", ErrUnsupportedSyntax)
	}

	return stmt, nil
}

// parseColumnList parses a comma-separated list of expressions.
func (p *Parser) parseColumnList() []Expression {
	cols := make([]Expression, 0)

	for {
		// Stop at FROM
		tok := p.peek()
		if tok.Type == TokenKeyword && tok.Value == "FROM" {
			break
		}

		expr, err := p.parseExpression()
		if err != nil {
			break
		}
		cols = append(cols, *expr)

		// Check for comma
		if p.peek().Type == TokenSymbol && p.peek().Value == "," {
			p.next()
			continue
		}
		break
	}

	return cols
}

// parseIdentifierList parses a comma-separated list of identifiers.
func (p *Parser) parseIdentifierList() []string {
	ids := make([]string, 0)

	for {
		tok := p.next()
		if tok.Type != TokenIdentifier {
			break
		}
		ids = append(ids, tok.Value)

		// Check for comma
		if p.peek().Type == TokenSymbol && p.peek().Value == "," {
			p.next()
			continue
		}
		break
	}

	return ids
}

// parseValueList parses a parenthesized list of values.
func (p *Parser) parseValueList() ([]Expression, error) {
	// Expect (
	if tok := p.next(); tok.Type != TokenSymbol || tok.Value != "(" {
		return nil, fmt.Errorf("%w: expected (", ErrSyntaxError)
	}

	values := make([]Expression, 0)

	for {
		// Check for )
		if p.peek().Type == TokenSymbol && p.peek().Value == ")" {
			p.next()
			break
		}

		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		values = append(values, *expr)

		// Check for comma
		if p.peek().Type == TokenSymbol && p.peek().Value == "," {
			p.next()
			continue
		}

		// Check for )
		if p.peek().Type == TokenSymbol && p.peek().Value == ")" {
			p.next()
			break
		}

		return nil, fmt.Errorf("%w: expected , or )", ErrSyntaxError)
	}

	return values, nil
}

// parseSetClause parses a SET clause.
func (p *Parser) parseSetClause() (*SetClause, error) {
	// Parse column name
	colTok := p.next()
	if colTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected column name", ErrSyntaxError)
	}

	// Expect =
	if tok := p.next(); tok.Type != TokenSymbol || tok.Value != "=" {
		return nil, fmt.Errorf("%w: expected =", ErrSyntaxError)
	}

	// Parse value
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &SetClause{
		Column: colTok.Value,
		Value:  *expr,
	}, nil
}

// parseJoin parses a JOIN clause.
func (p *Parser) parseJoin() (*JoinClause, error) {
	// Skip the first token (could be JOIN, INNER, LEFT, RIGHT, CROSS)
	firstTok := p.next()

	// Check if the first token was a join type keyword
	joinType := ""
	if firstTok.Type == TokenKeyword {
		if firstTok.Value == "INNER" || firstTok.Value == "LEFT" || firstTok.Value == "RIGHT" || firstTok.Value == "CROSS" {
			joinType = firstTok.Value
			// Now the next token should be JOIN
			tok := p.next()
			if tok.Type == TokenSymbol || tok.Value != "JOIN" {
				// Put it back if it's not JOIN
				p.tokPos--
			}
		}
	}

	// If no explicit join type, default to INNER
	if joinType == "" {
		joinType = "INNER"
		// If first token was just "JOIN", we're done, otherwise put it back
		if firstTok.Value != "JOIN" {
			p.tokPos--
		}
	}

	// Parse table name
	tableTok := p.next()
	if tableTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected table name", ErrSyntaxError)
	}

	join := &JoinClause{
		Type:  joinType,
		Table: tableTok.Value,
	}

	// Parse optional AS alias
	if p.peek().Type == TokenKeyword && p.peek().Value == "AS" {
		p.next()
		aliasTok := p.next()
		if aliasTok.Type == TokenIdentifier {
			join.Alias = aliasTok.Value
		}
	}

	// Parse ON condition
	if p.peek().Type == TokenKeyword && p.peek().Value == "ON" {
		p.next()
		cond, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		join.Condition = *cond
	}

	return join, nil
}

// parseOrderBy parses ORDER BY clause.
func (p *Parser) parseOrderBy() []OrderByClause {
	orderBy := make([]OrderByClause, 0)

	for {
		expr, err := p.parseExpression()
		if err != nil {
			break
		}

		clause := OrderByClause{Column: *expr}

		// Check for ASC/DESC
		if p.peek().Type == TokenKeyword {
			tok := p.next()
			if tok.Value == "DESC" {
				clause.Desc = true
			} else if tok.Value != "ASC" {
				// Put it back
				p.tokPos--
			}
		}

		orderBy = append(orderBy, clause)

		// Check for comma
		if p.peek().Type == TokenSymbol && p.peek().Value == "," {
			p.next()
			continue
		}
		break
	}

	return orderBy
}

// parseGroupBy parses GROUP BY clause.
func (p *Parser) parseGroupBy() []Expression {
	groupBy := make([]Expression, 0)

	for {
		expr, err := p.parseExpression()
		if err != nil {
			break
		}
		groupBy = append(groupBy, *expr)

		// Check for comma
		if p.peek().Type == TokenSymbol && p.peek().Value == "," {
			p.next()
			continue
		}
		break
	}

	return groupBy
}

// parseColumnDefinition parses a column definition.
func (p *Parser) parseColumnDefinition() (*ColumnDefinition, error) {
	col := &ColumnDefinition{}

	// Parse column name
	nameTok := p.next()
	if nameTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected column name", ErrSyntaxError)
	}
	col.Name = nameTok.Value

	// Parse column type
	typeTok := p.next()
	if typeTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("%w: expected column type", ErrSyntaxError)
	}
	col.Type = typeTok.Value

	// Parse constraints
	for {
		tok := p.peek()
		if tok.Type != TokenKeyword {
			break
		}

		switch tok.Value {
		case "PRIMARY", "KEY":
			p.next()
			col.PrimaryKey = true
		case "NOT", "NULL":
			p.next()
			if p.peek().Type == TokenKeyword && p.peek().Value == "NULL" {
				p.next()
			}
			col.NotNull = true
		case "UNIQUE":
			p.next()
			col.Unique = true
		case "AUTOINCREMENT":
			p.next()
			col.AutoInc = true
		case "DEFAULT":
			p.next()
			defaultVal, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			col.Default = *defaultVal
		default:
			break
		}
	}

	return col, nil
}

// parseExpression parses an expression.
func (p *Parser) parseExpression() (*Expression, error) {
	return p.parseOr()
}

// parseOr parses OR expressions.
func (p *Parser) parseOr() (*Expression, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == TokenKeyword && p.peek().Value == "OR" {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &Expression{
			Type:  ExprBinary,
			Op:    "OR",
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseAnd parses AND expressions.
func (p *Parser) parseAnd() (*Expression, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == TokenKeyword && p.peek().Value == "AND" {
		p.next()
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &Expression{
			Type:  ExprBinary,
			Op:    "AND",
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseEquality parses equality expressions.
func (p *Parser) parseEquality() (*Expression, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.peek()

		// Check for IS [NOT] NULL
		if tok.Type == TokenKeyword && tok.Value == "IS" {
			p.next() // Skip IS
			// Check for NOT
			if p.peek().Type == TokenKeyword && p.peek().Value == "NOT" {
				p.next()
			}
			// Expect NULL
			if p.peek().Type != TokenKeyword || p.peek().Value != "NULL" {
				return nil, fmt.Errorf("%w: expected NULL", ErrSyntaxError)
			}
			p.next() // Skip NULL
			return &Expression{
				Type:  ExprBinary,
				Op:    "IS",
				Left:  left,
				Right: &Expression{Type: ExprLiteral, Value: nil},
			}, nil
		}

		if tok.Type != TokenSymbol {
			break
		}

		var op string
		switch tok.Value {
		case "=":
			op = "="
		case "!=", "<>":
			op = "<>"
		default:
			break
		}

		if op == "" {
			break
		}

		p.next()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &Expression{
			Type:  ExprBinary,
			Op:    op,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseComparison parses comparison expressions.
func (p *Parser) parseComparison() (*Expression, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.peek()
		if tok.Type != TokenSymbol {
			break
		}

		var op string
		switch tok.Value {
		case ">":
			op = ">"
		case "<":
			op = "<"
		case ">=":
			op = ">="
		case "<=":
			op = "<="
		default:
			break
		}

		if op == "" {
			break
		}

		p.next()
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		left = &Expression{
			Type:  ExprBinary,
			Op:    op,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseAddSub parses addition and subtraction.
func (p *Parser) parseAddSub() (*Expression, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.peek()
		if tok.Type != TokenSymbol {
			break
		}

		if tok.Value != "+" && tok.Value != "-" {
			break
		}

		op := tok.Value
		p.next()
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = &Expression{
			Type:  ExprBinary,
			Op:    op,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseMulDiv parses multiplication and division.
func (p *Parser) parseMulDiv() (*Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.peek()
		if tok.Type != TokenSymbol {
			break
		}

		if tok.Value != "*" && tok.Value != "/" {
			break
		}

		op := tok.Value
		p.next()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = &Expression{
			Type:  ExprBinary,
			Op:    op,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parsePrimary parses primary expressions.
func (p *Parser) parsePrimary() (*Expression, error) {
	tok := p.next()

	switch tok.Type {
	case TokenNumber:
		// Try to parse as integer first
		if val, err := strconv.ParseInt(tok.Value, 10, 64); err == nil {
			return &Expression{
				Type:  ExprLiteral,
				Value: val,
			}, nil
		}
		// Try as float
		if val, err := strconv.ParseFloat(tok.Value, 64); err == nil {
			return &Expression{
				Type:  ExprLiteral,
				Value: val,
			}, nil
		}
		return nil, fmt.Errorf("%w: invalid number: %s", ErrInvalidValue, tok.Value)

	case TokenString:
		return &Expression{
			Type:  ExprLiteral,
			Value: tok.Value,
		}, nil

	case TokenKeyword:
		switch tok.Value {
		case "TRUE":
			return &Expression{
				Type:  ExprLiteral,
				Value: true,
			}, nil
		case "FALSE":
			return &Expression{
				Type:  ExprLiteral,
				Value: false,
			}, nil
		case "NULL":
			return &Expression{
				Type:  ExprLiteral,
				Value: nil,
			}, nil
		}
		// Keywords can be column names
		return &Expression{
			Type:  ExprColumn,
			Value: tok.Value,
		}, nil

	case TokenIdentifier:
		// Check for table.column notation
		if p.peek().Type == TokenSymbol && p.peek().Value == "." {
			p.next() // Skip .
			table := tok.Value
			colTok := p.next()
			if colTok.Type != TokenIdentifier {
				return nil, fmt.Errorf("%w: expected column name", ErrSyntaxError)
			}
			return &Expression{
				Type:  ExprColumn,
				Value: table + "." + colTok.Value,
			}, nil
		}

		// Check for function call (identifier followed by ()
		if p.peek().Type == TokenSymbol && p.peek().Value == "(" {
			funcName := tok.Value
			p.next() // Skip (
			// Parse arguments
			args := make([]Expression, 0)
			for p.peek().Type != TokenSymbol || p.peek().Value != ")" {
				arg, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				args = append(args, *arg)
				if p.peek().Type == TokenSymbol && p.peek().Value == "," {
					p.next()
					continue
				}
				break
			}
			p.next() // Skip )
			return &Expression{
				Type:  ExprFunction,
				Value: funcName,
				Left:  nil,
				Right: nil,
			}, nil
		}

		return &Expression{
			Type:  ExprColumn,
			Value: tok.Value,
		}, nil

	case TokenSymbol:
		if tok.Value == "(" {
			expr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if tok := p.next(); tok.Type != TokenSymbol || tok.Value != ")" {
				return nil, fmt.Errorf("%w: expected )", ErrSyntaxError)
			}
			return expr, nil
		}
		if tok.Value == "*" {
			return &Expression{
				Type:  ExprColumn,
				Value: "*",
			}, nil
		}
	}

	return nil, fmt.Errorf("%w: %s at position %d", ErrUnexpectedToken, tok.Value, tok.Pos)
}

// ParseSQL parses a SQL statement.
func ParseSQL(sql string) (*Statement, error) {
	parser := NewParser(sql)
	return parser.Parse()
}
