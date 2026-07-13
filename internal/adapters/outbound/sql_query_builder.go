package outbound

import (
	"database/sql"
	"fmt"
	"strings"

	"fsos-server/internal/domain/ports"
)

// dbExecutor is the subset of *sql.DB used by queries. Both *sql.DB and *sql.Tx
// satisfy it, so a query builder can run against the root connection or inside
// a transaction with no behavioral difference.
type dbExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// sqlQueryBuilder implements ports.QueryBuilder (and, over a *sql.DB,
// ports.TransactionalQueryBuilder) for any backend: queries are built with
// ?-placeholders and rebound per dialect at execution time.
type sqlQueryBuilder struct {
	exec    dbExecutor
	dialect dialect
}

func (qb *sqlQueryBuilder) Table(name string) ports.Query {
	return &sqlQuery{tableName: name, exec: qb.exec, dialect: qb.dialect}
}

// Begin opens a transaction and returns a ports.Tx whose queries run inside it.
// Only valid on a builder backed by the root *sql.DB (not an already-tx one).
func (qb *sqlQueryBuilder) Begin() (ports.Tx, error) {
	db, ok := qb.exec.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("cannot begin transaction: query builder is not backed by a root connection")
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx: tx, dialect: qb.dialect}, nil
}

// sqlTx implements ports.Tx: a query builder bound to a *sql.Tx.
type sqlTx struct {
	tx      *sql.Tx
	dialect dialect
}

func (t *sqlTx) Table(name string) ports.Query {
	return &sqlQuery{tableName: name, exec: t.tx, dialect: t.dialect}
}

func (t *sqlTx) Commit() error   { return t.tx.Commit() }
func (t *sqlTx) Rollback() error { return t.tx.Rollback() }

type whereClause struct {
	column string
	value  interface{}
}

// sqlQuery implements ports.Query over a dialect.
type sqlQuery struct {
	tableName string
	wheres    []whereClause
	exec      dbExecutor
	dialect   dialect
}

func (q *sqlQuery) Where(column string, value interface{}) ports.Query {
	newWheres := make([]whereClause, len(q.wheres), len(q.wheres)+1)
	copy(newWheres, q.wheres)
	newWheres = append(newWheres, whereClause{column, value})
	return &sqlQuery{
		tableName: q.tableName,
		wheres:    newWheres,
		exec:      q.exec,
		dialect:   q.dialect,
	}
}

func (q *sqlQuery) Insert(data map[string]interface{}) error {
	if err := validateIdentifier(q.tableName); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data))
	for col, val := range data {
		if err := validateIdentifier(col); err != nil {
			return fmt.Errorf("invalid column name %q: %w", col, err)
		}
		cols = append(cols, col)
		placeholders = append(placeholders, "?")
		vals = append(vals, val)
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		q.tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	_, err := q.exec.Exec(q.dialect.Rebind(query), vals...)
	return err
}

func (q *sqlQuery) Update(data map[string]interface{}) (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	sets := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data)+len(q.wheres))
	for col, val := range data {
		if err := validateIdentifier(col); err != nil {
			return 0, fmt.Errorf("invalid column name %q: %w", col, err)
		}
		sets = append(sets, col+" = ?")
		vals = append(vals, val)
	}
	whereSQL, whereVals, err := q.buildWhere()
	if err != nil {
		return 0, err
	}
	vals = append(vals, whereVals...)

	query := fmt.Sprintf("UPDATE %s SET %s%s", q.tableName, strings.Join(sets, ", "), whereSQL)
	result, execErr := q.exec.Exec(q.dialect.Rebind(query), vals...)
	if execErr != nil {
		return 0, execErr
	}
	return result.RowsAffected()
}

func (q *sqlQuery) Increment(column string, delta int64) (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	if err := validateIdentifier(column); err != nil {
		return 0, fmt.Errorf("invalid column name %q: %w", column, err)
	}
	vals := []interface{}{delta}
	whereSQL, whereVals, err := q.buildWhere()
	if err != nil {
		return 0, err
	}
	vals = append(vals, whereVals...)

	query := fmt.Sprintf("UPDATE %s SET %s = %s + ?%s", q.tableName, column, column, whereSQL)
	result, execErr := q.exec.Exec(q.dialect.Rebind(query), vals...)
	if execErr != nil {
		return 0, execErr
	}
	return result.RowsAffected()
}

func (q *sqlQuery) Delete() (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere()
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("DELETE FROM %s%s", q.tableName, whereSQL)
	result, execErr := q.exec.Exec(q.dialect.Rebind(query), vals...)
	if execErr != nil {
		return 0, execErr
	}
	return result.RowsAffected()
}

func (q *sqlQuery) First() (ports.QueryResult, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere()
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s%s LIMIT 1", q.tableName, whereSQL)
	rows, err := q.exec.Query(q.dialect.Rebind(query), vals...)
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

func (q *sqlQuery) Get() ([]ports.QueryResult, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere()
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s%s", q.tableName, whereSQL)
	rows, err := q.exec.Query(q.dialect.Rebind(query), vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (q *sqlQuery) Count() (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere()
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", q.tableName, whereSQL)
	var count int64
	err = q.exec.QueryRow(q.dialect.Rebind(query), vals...).Scan(&count)
	return count, err
}

func (q *sqlQuery) buildWhere() (string, []interface{}, error) {
	if len(q.wheres) == 0 {
		return "", nil, nil
	}
	parts := make([]string, len(q.wheres))
	vals := make([]interface{}, len(q.wheres))
	for i, w := range q.wheres {
		if err := validateIdentifier(w.column); err != nil {
			return "", nil, fmt.Errorf("invalid column name %q: %w", w.column, err)
		}
		parts[i] = w.column + " = ?"
		vals[i] = w.value
	}
	return " WHERE " + strings.Join(parts, " AND "), vals, nil
}

func scanRows(rows *sql.Rows) ([]ports.QueryResult, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var results []ports.QueryResult
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(ports.QueryResult, len(cols))
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
