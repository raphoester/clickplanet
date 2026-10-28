package appearances_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// A post scheduled is not out yet. Its address on a platform is the newest snapshot's that has one.
const appearing = `
	SELECT a.post_id, a.title, a.scheduled_at, p.platform::text, COALESCE(r.url, '')
	FROM appearances a
	CROSS JOIN LATERAL unnest(a.platforms) WITH ORDINALITY AS p(platform, place)
	LEFT JOIN LATERAL (
		SELECT s.document->>'url' AS url
		FROM snapshot_rows s
		WHERE s.post_id = a.post_id AND s.platform = p.platform::text AND COALESCE(s.document->>'url', '') <> ''
		ORDER BY s.taken_at DESC
		LIMIT 1
	) r ON true
	WHERE a.account = $1 AND a.scheduled_at <= $2 AND a.scheduled_at >= $3
	ORDER BY a.scheduled_at DESC, a.post_id, p.place
`

func (q *PostgresQuery) Appearances(ctx context.Context, account casts.AccountID) (*clipsv1.GetAppearancesResponse, error) {
	now := q.clock.Now()
	rows, err := q.db.On(ctx).QueryContext(ctx, appearing, uuid.UUID(account), now, appearances.ShownSince(now))
	if err != nil {
		return nil, fmt.Errorf("failed to read the posts that name the player: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &clipsv1.GetAppearancesResponse{}
	var last *clipsv1.Appearance
	for rows.Next() {
		var (
			post, title, platform, url string
			scheduledAt                time.Time
		)
		if err := rows.Scan(&post, &title, &scheduledAt, &platform, &url); err != nil {
			return nil, fmt.Errorf("failed to read a post that names the player: %w", err)
		}
		if last == nil || last.GetPostId() != post {
			last = &clipsv1.Appearance{PostId: post, Title: title, ScheduledAt: timestamppb.New(scheduledAt)}
			answer.Appearances = append(answer.Appearances, last)
		}
		last.Links = append(last.Links, &clipsv1.ClipLink{Platform: platform, Url: url})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the posts that name the player: %w", err)
	}

	if err := q.db.On(ctx).QueryRowContext(ctx, `SELECT NOT EXISTS (SELECT 1 FROM opt_outs WHERE account = $1)`,
		uuid.UUID(account)).Scan(&answer.NameShown); err != nil {
		return nil, fmt.Errorf("failed to read whether the player keeps out of the clips: %w", err)
	}
	return answer, nil
}
