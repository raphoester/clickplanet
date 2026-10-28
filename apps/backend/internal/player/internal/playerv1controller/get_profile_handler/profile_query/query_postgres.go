package profile_query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Renames interface {
	RenamableAt(chosenAt time.Time) int64
}

func NewPostgresQuery(db cppg.Conn, renames Renames) *PostgresQuery {
	return &PostgresQuery{db: db, renames: renames}
}

type PostgresQuery struct {
	db      cppg.Conn
	renames Renames
}

const profile = `
	SELECT COALESCE(profiles.name, ''), COALESCE(profiles.color, 0), profiles.name_chosen_at,
		COALESCE(profiles.shown_in_clips, false)
	FROM (SELECT $1::uuid AS account_id) asked
	LEFT JOIN profiles ON profiles.account_id = asked.account_id
`

func (q *PostgresQuery) Profile(ctx context.Context, account cpsession.AccountID) (*playerv1.GetProfileResponse, error) {
	var (
		name         string
		color        int32
		chosenAt     sql.NullTime
		shownInClips bool
	)
	if err := q.db.On(ctx).QueryRowContext(ctx, profile, uuid.UUID(account)).Scan(&name, &color, &chosenAt, &shownInClips); err != nil {
		return nil, fmt.Errorf("failed to read the profile: %w", err)
	}

	kept, err := playerread.KeptColor(color)
	if err != nil {
		return nil, fmt.Errorf("failed to read the profile's color: %w", err)
	}

	var chosen time.Time
	if chosenAt.Valid {
		chosen = chosenAt.Time
	}

	return &playerv1.GetProfileResponse{
		Profile:      &playerv1.Profile{AccountId: account.String(), Name: name, RenamableAtUnixMs: q.renames.RenamableAt(chosen)},
		Color:        kept,
		ShownInClips: shownInClips,
	}, nil
}
