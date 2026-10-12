package statement_query

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	opsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	MaxLimit = 20000

	usualLimit = 1000
	maxBytes   = 8 << 20

	// A JSON number is a float64: past this an integer would come back as its neighbour.
	maxExactInteger = 1 << 53

	queryCanceled = "57014"
)

var (
	ErrNoStatement  = errors.New("the statement is empty")
	ErrLimitTooHigh = fmt.Errorf("the limit is over %d", MaxLimit)
	ErrRefused      = errors.New("postgres refused the statement")
	ErrTooSlow      = errors.New("the statement took too long")
	ErrUnreachable  = errors.New("postgres cannot be read")
)

func NewPostgresQuery(db cppg.Beginner, timeout time.Duration) PostgresQuery {
	return PostgresQuery{db: db, timeout: timeout}
}

type PostgresQuery struct {
	db      cppg.Beginner
	timeout time.Duration
}

func (q PostgresQuery) Rows(ctx context.Context, statement string, limit uint32) (*opsv1.QueryResponse, error) {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return nil, ErrNoStatement
	}
	if limit > MaxLimit {
		return nil, ErrLimitTooHigh
	}
	if limit == 0 {
		limit = usualLimit
	}

	ctx, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()

	tx, err := q.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, unreachable(err)
	}
	defer func() { _ = tx.Rollback() }()

	// Prepared, so postgres refuses a string that holds more than one statement.
	prepared, err := tx.PrepareContext(ctx, statement)
	if err != nil {
		return nil, named(ctx, err)
	}
	defer func() { _ = prepared.Close() }()

	rows, err := prepared.QueryContext(ctx)
	if err != nil {
		return nil, named(ctx, err)
	}
	defer func() { _ = rows.Close() }()

	response, err := read(rows, int(limit))
	if err != nil {
		return nil, named(ctx, err)
	}

	return response, nil
}

func read(rows *sql.Rows, limit int) (*opsv1.QueryResponse, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to read the column types: %w", err)
	}

	response := &opsv1.QueryResponse{Columns: make([]string, len(types))}
	for i, column := range types {
		response.Columns[i] = column.Name()
	}

	size := 0
	for rows.Next() {
		if len(response.GetRows()) >= limit || size > maxBytes {
			response.Truncated = true
			break
		}

		cells := make([]any, len(types))
		targets := make([]any, len(types))
		for i := range cells {
			targets[i] = &cells[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return nil, fmt.Errorf("failed to scan a row: %w", err)
		}

		row := &structpb.ListValue{Values: make([]*structpb.Value, len(cells))}
		for i, cell := range cells {
			row.Values[i] = valueOf(cell, types[i].DatabaseTypeName())
		}
		size += proto.Size(row)
		response.Rows = append(response.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rows: %w", err)
	}

	return response, nil
}

func valueOf(cell any, databaseType string) *structpb.Value {
	switch typed := cell.(type) {
	case nil:
		return structpb.NewNullValue()
	case bool:
		return structpb.NewBoolValue(typed)
	case int64:
		if typed > maxExactInteger || typed < -maxExactInteger {
			return structpb.NewStringValue(strconv.FormatInt(typed, 10))
		}
		return structpb.NewNumberValue(float64(typed))
	case float64:
		// JSON has no NaN and no infinity.
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return structpb.NewStringValue(strconv.FormatFloat(typed, 'g', -1, 64))
		}
		return structpb.NewNumberValue(typed)
	case time.Time:
		return structpb.NewStringValue(typed.Format(time.RFC3339Nano))
	case []byte:
		if databaseType == "BYTEA" {
			return structpb.NewStringValue(`\x` + hex.EncodeToString(typed))
		}
		return textOf(string(typed))
	case string:
		return textOf(typed)
	default:
		return textOf(fmt.Sprint(typed))
	}
}

// A proto string must be UTF-8, and one stray byte would fail the whole answer.
func textOf(text string) *structpb.Value {
	return structpb.NewStringValue(strings.ToValidUTF8(text, "�"))
}

func unreachable(err error) error {
	var refused *pq.Error
	if errors.As(err, &refused) {
		return fmt.Errorf("%w: %s", ErrUnreachable, refused.Message)
	}

	return fmt.Errorf("%w: %w", ErrUnreachable, err)
}

func named(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ErrTooSlow
	}

	var refused *pq.Error
	if !errors.As(err, &refused) {
		return err
	}
	if refused.Code == queryCanceled {
		return fmt.Errorf("%w: %s", ErrRefused, refused.Message)
	}

	switch refused.Code.Class() {
	case "08", "28", "53", "57", "58", "XX":
		return fmt.Errorf("%w: %s", ErrUnreachable, refused.Message)
	default:
		return fmt.Errorf("%w: %s", ErrRefused, refused.Message)
	}
}
