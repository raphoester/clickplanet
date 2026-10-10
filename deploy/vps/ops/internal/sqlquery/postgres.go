package sqlquery

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/lib/pq"
)

func OpenPostgres(url string, timeout time.Duration) (*Postgres, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to read the postgres url: %w", err)
	}
	db.SetMaxOpenConns(2)
	// No connection outlives its query, so a session lock or setting cannot either.
	db.SetMaxIdleConns(0)
	return &Postgres{db: db, timeout: timeout}, nil
}

type Postgres struct {
	db      *sql.DB
	timeout time.Duration
}

func (p *Postgres) Close() error {
	return p.db.Close()
}

func (p *Postgres) Execute(ctx context.Context, query Query) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Result{}, fmt.Errorf("failed to begin a read-only transaction: %w", refusal(err))
	}
	defer func() { _ = tx.Rollback() }()

	// Prepared, so postgres refuses a string that holds more than one statement.
	statement, err := tx.PrepareContext(ctx, query.Statement)
	if err != nil {
		return Result{}, fmt.Errorf("failed to prepare the statement: %w", refusal(err))
	}
	defer func() { _ = statement.Close() }()

	rows, err := statement.QueryContext(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("failed to run the statement: %w", refusal(err))
	}
	defer func() { _ = rows.Close() }()

	result, err := collect(rows, query)
	if err != nil {
		return Result{}, refusal(err)
	}
	return result, nil
}

func collect(rows *sql.Rows, query Query) (Result, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return Result{}, fmt.Errorf("failed to read the column types: %w", err)
	}

	result := Result{Columns: make([]string, len(types)), Rows: [][]any{}}
	for i, column := range types {
		result.Columns[i] = column.Name()
	}

	size := 0
	for rows.Next() {
		if len(result.Rows) >= query.MaxRows || size > query.MaxBytes {
			result.Truncated = true
			break
		}

		values := make([]any, len(types))
		targets := make([]any, len(types))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return Result{}, fmt.Errorf("failed to scan a row: %w", err)
		}

		for i, value := range values {
			values[i] = printable(value, types[i].DatabaseTypeName())
			size += weight(values[i])
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return Result{}, fmt.Errorf("failed to read the rows: %w", err)
	}
	return result, nil
}

func printable(value any, databaseType string) any {
	switch typed := value.(type) {
	case []byte:
		if databaseType == "BYTEA" {
			return `\x` + hex.EncodeToString(typed)
		}
		return string(typed)
	case float64:
		// JSON has no NaN and no infinity.
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return strconv.FormatFloat(typed, 'g', -1, 64)
		}
		return typed
	default:
		return value
	}
}

func weight(value any) int {
	if text, isText := value.(string); isText {
		return len(text)
	}
	return 8
}

func refusal(err error) error {
	var refused *pq.Error
	if errors.As(err, &refused) {
		return fmt.Errorf("%w: %s", ErrRefused, refused.Message)
	}
	return err
}
