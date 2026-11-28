package flags_query

import (
	"context"
	"errors"
	"fmt"
	"time"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrUnknownState = errors.New("a flag kept in a state this build does not know")

func NewPostgresQuery(db cppg.Conn, messages flagmessage.Messages) PostgresQuery {
	return PostgresQuery{db: db, messages: messages}
}

type PostgresQuery struct {
	db       cppg.Conn
	messages flagmessage.Messages
}

func (q PostgresQuery) Flags(ctx context.Context) (*flagsv1.ListFlagsResponse, error) {
	builtIn := cpflags.BuiltIn()
	flags := make([]*flagsv1.Flag, 0, len(builtIn))
	for _, flag := range builtIn {
		flags = append(flags, &flagsv1.Flag{Code: flag.Code, Name: flag.Name})
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT id, name, state::text, decided_at, ends_at,
			picture_hash, picture_width, picture_height, picture_stretch, picture_focus_x, picture_focus_y,
			pin_latitude, pin_longitude, COALESCE(anthem_library, '')
		FROM flags ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flags opened: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var code, name, state, anthem string
		var decidedAt, endsAt time.Time
		var picture pictures.Snapshot
		var focusX, focusY float32
		var pin catalog.Pin
		if err := rows.Scan(&code, &name, &state, &decidedAt, &endsAt,
			&picture.Hash, &picture.Width, &picture.Height, &picture.Fit.Stretch, &focusX, &focusY,
			&pin.Latitude, &pin.Longitude, &anthem); err != nil {
			return nil, fmt.Errorf("failed to read a flag opened: %w", err)
		}
		kept, err := stateOf(state)
		if err != nil {
			return nil, fmt.Errorf("the flag %s: %w", code, err)
		}
		picture.Fit.FocusX, picture.Fit.FocusY = float64(focusX), float64(focusY)
		message := q.messages.Flag(catalog.Snapshot{
			ID: catalog.FlagID(code), Name: catalog.Name(name), Picture: picture, Pin: pin, State: kept,
			DecidedAt: decidedAt, EndsAt: endsAt, Anthem: anthem,
		})
		flags = append(flags, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the flags opened: %w", err)
	}
	return &flagsv1.ListFlagsResponse{Flags: flags}, nil
}

func stateOf(state string) (catalog.State, error) {
	switch kept := catalog.State(state); kept {
	case catalog.Open, catalog.Closed, catalog.Removed:
		return kept, nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownState, state)
}
