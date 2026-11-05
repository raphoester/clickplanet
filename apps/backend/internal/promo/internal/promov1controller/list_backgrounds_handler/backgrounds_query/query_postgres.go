package backgrounds_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

type Links interface {
	PublicURL(id backgrounds.BackgroundID) string
}

func NewPostgresQuery(db cppg.Conn, links Links) *PostgresQuery {
	return &PostgresQuery{db: db, links: links}
}

type PostgresQuery struct {
	db    cppg.Conn
	links Links
}

// The newest first.
func (q *PostgresQuery) Backgrounds(ctx context.Context) (*promov1.ListBackgroundsResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT background_id, name, media_type, size_bytes, kept_at
		FROM backgrounds
		ORDER BY kept_at DESC, background_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the backgrounds: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := &promov1.ListBackgroundsResponse{}
	for rows.Next() {
		var (
			id     uuid.UUID
			name   string
			media  string
			size   int64
			keptAt time.Time
		)
		if err := rows.Scan(&id, &name, &media, &size, &keptAt); err != nil {
			return nil, fmt.Errorf("failed to read a background: %w", err)
		}
		out.Backgrounds = append(out.Backgrounds, &promov1.Background{
			BackgroundId: id.String(), Name: name, ContentType: media, SizeBytes: uint64(max(size, 0)),
			KeptAt: timestamppb.New(keptAt), Url: q.links.PublicURL(backgrounds.BackgroundID(id)),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the backgrounds: %w", err)
	}
	return out, nil
}
