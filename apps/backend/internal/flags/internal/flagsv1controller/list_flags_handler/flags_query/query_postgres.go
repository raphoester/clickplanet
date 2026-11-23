package flags_query

import (
	"context"
	"errors"
	"fmt"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrUnknownState = errors.New("a flag kept in a state this build does not know")

func NewPostgresQuery(db cppg.Conn) PostgresQuery {
	return PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q PostgresQuery) Flags(ctx context.Context) (*flagsv1.ListFlagsResponse, error) {
	builtIn := cpflags.BuiltIn()
	flags := make([]*flagsv1.Flag, 0, len(builtIn))
	for _, flag := range builtIn {
		flags = append(flags, &flagsv1.Flag{Code: flag.Code, Name: flag.Name})
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, `SELECT id, name, state::text FROM flags ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flags opened: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var code, name, state string
		if err := rows.Scan(&code, &name, &state); err != nil {
			return nil, fmt.Errorf("failed to read a flag opened: %w", err)
		}
		closed, err := closedOf(state)
		if err != nil {
			return nil, fmt.Errorf("the flag %s: %w", code, err)
		}
		flags = append(flags, &flagsv1.Flag{Code: code, Name: name, Closed: closed})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the flags opened: %w", err)
	}
	return &flagsv1.ListFlagsResponse{Flags: flags}, nil
}

func closedOf(state string) (bool, error) {
	switch state {
	case "open":
		return false, nil
	case "closed":
		return true, nil
	}
	return false, fmt.Errorf("%w: %q", ErrUnknownState, state)
}
