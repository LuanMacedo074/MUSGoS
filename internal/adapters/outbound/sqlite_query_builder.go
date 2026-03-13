package outbound

import (
	"database/sql"
	"fmt"
	"strings"

	"fsos-server/internal/domain/ports"
)

// SQLiteQueryBuilder implements ports.QueryBuilder for SQLite.
type SQLiteQueryBuilder struct {
	db *sql.DB
}

func NewSQLiteQueryBuilder(db *sql.DB) *SQLiteQueryBuilder {
	return &SQLiteQueryBuilder{db: db}
}

func (qb *SQLiteQueryBuilder) Table(name string) ports.Query {
	return &sqliteQuery{
		tableName: name,
		db:        qb.db,
	}
}

type whereClause struct {
	column string
	value  interface{}
}

// sqliteQuery implements ports.Query for SQLite.
type sqliteQuery struct {
	tableName string
	wheres    []whereClause
	db        *sql.DB
}

func (q *sqliteQuery) Where(column string, value interface{}) ports.Query {
	newWheres := make([]whereClause, len(q.wheres), len(q.wheres)+1)
	copy(newWheres, q.wheres)
	newWheres = append(newWheres, whereClause{column, value})
	return &sqliteQuery{
		tableName: q.tableName,
		wheres:    newWheres,
		db:        q.db,
	}
}

func (q *sqliteQuery) Insert(data map[string]interface{}) error {
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
	_, err := q.db.Exec(query, vals...)
	return err
}

func (q *sqliteQuery) Update(data map[string]interface{}) (int64, error) {
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
	result, execErr := q.db.Exec(query, vals...)
	if execErr != nil {
		return 0, execErr
	}
	return result.RowsAffected()
}

func (q *sqliteQuery) Delete() (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere()
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("DELETE FROM %s%s", q.tableName, whereSQL)
	result, execErr := q.db.Exec(query, vals...)
	if execErr != nil {
		return 0, execErr
	}
	return result.RowsAffected()
}

func (q *sqliteQuery) First() (ports.QueryResult, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere()
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s%s LIMIT 1", q.tableName, whereSQL)
	rows, err := q.db.Query(query, vals...)
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

func (q *sqliteQuery) Get() ([]ports.QueryResult, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere()
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s%s", q.tableName, whereSQL)
	rows, err := q.db.Query(query, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (q *sqliteQuery) Count() (int64, error) {
	if err := validateIdentifier(q.tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	whereSQL, vals, err := q.buildWhere()
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", q.tableName, whereSQL)
	var count int64
	err = q.db.QueryRow(query, vals...).Scan(&count)
	return count, err
}

func (q *sqliteQuery) buildWhere() (string, []interface{}, error) {
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
