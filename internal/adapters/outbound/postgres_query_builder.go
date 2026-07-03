package outbound

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"fsos-server/internal/domain/ports"
)

// PostgresQueryBuilder implements ports.QueryBuilder (and, over a *sql.DB,
// ports.TransactionalQueryBuilder) for PostgreSQL. It mirrors
// SQLiteQueryBuilder but emits $N placeholders instead of ?.
type PostgresQueryBuilder struct {
	exec dbExecutor
}

func NewPostgresQueryBuilder(db *sql.DB) *PostgresQueryBuilder {
	return &PostgresQueryBuilder{exec: db}
}

func (qb *PostgresQueryBuilder) Table(name string) ports.Query {
	return &pgQuery{tableName: name, exec: qb.exec}
}

// Begin opens a transaction and returns a ports.Tx whose queries run inside it.
// Only valid on a builder backed by the root *sql.DB (not an already-tx one).
func (qb *PostgresQueryBuilder) Begin() (ports.Tx, error) {
	db, ok := qb.exec.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("cannot begin transaction: query builder is not backed by a root connection")
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return &pgTx{tx: tx}, nil
}

// pgTx implements ports.Tx: a query builder bound to a *sql.Tx.
type pgTx struct {
	tx *sql.Tx
}

func (t *pgTx) Table(name string) ports.Query {
	return &pgQuery{tableName: name, exec: t.tx}
}

func (t *pgTx) Commit() error   { return t.tx.Commit() }
func (t *pgTx) Rollback() error { return t.tx.Rollback() }

// pgQuery implements ports.Query for PostgreSQL.
type pgQuery struct {
	tableName string
	wheres    []whereClause
	exec      dbExecutor
}

func (q *pgQuery) Where(column string, value interface{}) ports.Query {
	newWheres := make([]whereClause, len(q.wheres), len(q.wheres)+1)
	copy(newWheres, q.wheres)
	newWheres = append(newWheres, whereClause{column, value})
	return &pgQuery{
		tableName: q.tableName,
		wheres:    newWheres,
		exec:      q.exec,
	}
}

func (q *pgQuery) Insert(data map[string]interface{}) error {
	if err := validateIdentifier(q.tableName); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data))
	n := 1
	for col, val := range data {
		if err := validateIdentifier(col); err != nil {
			return fmt.Errorf("invalid column name %q: %w", col, err)
		}
		cols = append(cols, col)
		placeholders = append(placeholders, "$"+strconv.Itoa(n))
		vals = append(vals, val)
		n++
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		q.tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	_, err := q.exec.Exec(query, vals...)
	return err
}

func (q *pgQuery) Update(data map[string]interface{}) (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	sets := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data)+len(q.wheres))
	n := 1
	for col, val := range data {
		if err := validateIdentifier(col); err != nil {
			return 0, fmt.Errorf("invalid column name %q: %w", col, err)
		}
		sets = append(sets, col+" = $"+strconv.Itoa(n))
		vals = append(vals, val)
		n++
	}
	whereSQL, whereVals, err := q.buildWhere(n)
	if err != nil {
		return 0, err
	}
	vals = append(vals, whereVals...)

	query := fmt.Sprintf("UPDATE %s SET %s%s", q.tableName, strings.Join(sets, ", "), whereSQL)
	result, execErr := q.exec.Exec(query, vals...)
	if execErr != nil {
		return 0, execErr
	}
	return result.RowsAffected()
}

func (q *pgQuery) Delete() (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere(1)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("DELETE FROM %s%s", q.tableName, whereSQL)
	result, execErr := q.exec.Exec(query, vals...)
	if execErr != nil {
		return 0, execErr
	}
	return result.RowsAffected()
}

func (q *pgQuery) First() (ports.QueryResult, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere(1)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s%s LIMIT 1", q.tableName, whereSQL)
	rows, err := q.exec.Query(query, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

func (q *pgQuery) Get() ([]ports.QueryResult, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere(1)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s%s", q.tableName, whereSQL)
	rows, err := q.exec.Query(query, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (q *pgQuery) Count() (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere(1)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", q.tableName, whereSQL)
	var count int64
	err = q.exec.QueryRow(query, vals...).Scan(&count)
	return count, err
}

// buildWhere renders the WHERE clause, numbering placeholders from start so
// they don't collide with any SET/VALUES placeholders emitted before it.
func (q *pgQuery) buildWhere(start int) (string, []interface{}, error) {
	if len(q.wheres) == 0 {
		return "", nil, nil
	}
	parts := make([]string, len(q.wheres))
	vals := make([]interface{}, len(q.wheres))
	for i, w := range q.wheres {
		if err := validateIdentifier(w.column); err != nil {
			return "", nil, fmt.Errorf("invalid column name %q: %w", w.column, err)
		}
		parts[i] = w.column + " = $" + strconv.Itoa(start+i)
		vals[i] = w.value
	}
	return " WHERE " + strings.Join(parts, " AND "), vals, nil
}
