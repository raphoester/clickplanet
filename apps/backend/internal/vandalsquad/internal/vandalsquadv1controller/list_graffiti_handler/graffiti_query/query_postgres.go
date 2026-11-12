package graffiti_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/protobuf/types/known/timestamppb"

	vandalsquadv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/vandalsquad/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const Listed = 500

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q *PostgresQuery) Graffiti(ctx context.Context) (*vandalsquadv1.ListGraffitiResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT account, flag, kind, tiles, flagged_at, seen_at, link
		FROM graffiti
		ORDER BY flagged_at DESC, id
		LIMIT $1
	`, Listed)
	if err != nil {
		return nil, fmt.Errorf("failed to list the graffiti: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &vandalsquadv1.ListGraffitiResponse{}
	for rows.Next() {
		var (
			account           uuid.NullUUID
			flag, kind, link  string
			tiles             pq.Int64Array
			flagged, lastSeen time.Time
		)
		if err := rows.Scan(&account, &flag, &kind, &tiles, &flagged, &lastSeen, &link); err != nil {
			return nil, fmt.Errorf("failed to read a graffiti: %w", err)
		}
		listedKind, err := kindOf(kind)
		if err != nil {
			return nil, err
		}
		listed := &vandalsquadv1.Graffiti{
			CountryId: flag, Kind: listedKind, TileIds: make([]uint32, len(tiles)),
			FlaggedAt: timestamppb.New(flagged), SeenAt: timestamppb.New(lastSeen), Link: link,
		}
		if account.Valid {
			listed.AccountId = account.UUID.String()
		}
		for i, tile := range tiles {
			listed.TileIds[i] = uint32(tile) //nolint:gosec // the map's ids, a few hundred thousand.
		}
		answer.Graffiti = append(answer.Graffiti, listed)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list the graffiti: %w", err)
	}
	return answer, nil
}

func kindOf(kind string) (vandalsquadv1.GraffitiKind, error) {
	switch kind {
	case "swastika":
		return vandalsquadv1.GraffitiKind_GRAFFITI_KIND_SWASTIKA, nil
	default:
		return vandalsquadv1.GraffitiKind_GRAFFITI_KIND_UNSPECIFIED, fmt.Errorf("postgres keeps a graffiti of kind %q", kind)
	}
}
