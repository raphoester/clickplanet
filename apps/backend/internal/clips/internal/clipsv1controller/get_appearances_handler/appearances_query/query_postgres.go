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

// A post on one platform.
type PostOnPlatform struct {
	Post     string
	Platform string
}

// Where each post is on each platform: the publishing module's daily snapshot read it (rpc_publishing_links). A post
// it has not read yet is not in the answer.
type Links interface {
	Of(ctx context.Context, posts []string) (map[PostOnPlatform]string, error)
}

func NewPostgresQuery(db cppg.Conn, links Links, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, links: links, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	links Links
	clock cptime.Clock
}

// A post scheduled is not out yet.
const appearing = `
	SELECT a.post_id, a.title, a.scheduled_at, p.platform::text
	FROM appearances a
	CROSS JOIN LATERAL unnest(a.platforms) WITH ORDINALITY AS p(platform, place)
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
			post, title, platform string
			scheduledAt           time.Time
		)
		if err := rows.Scan(&post, &title, &scheduledAt, &platform); err != nil {
			return nil, fmt.Errorf("failed to read a post that names the player: %w", err)
		}
		if last == nil || last.GetPostId() != post {
			last = &clipsv1.Appearance{PostId: post, Title: title, ScheduledAt: timestamppb.New(scheduledAt)}
			answer.Appearances = append(answer.Appearances, last)
		}
		last.Links = append(last.Links, &clipsv1.ClipLink{Platform: platform})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the posts that name the player: %w", err)
	}
	if err := q.withAddresses(ctx, answer); err != nil {
		return nil, err
	}

	if err := q.db.On(ctx).QueryRowContext(ctx, `SELECT NOT EXISTS (SELECT 1 FROM opt_outs WHERE account = $1)`,
		uuid.UUID(account)).Scan(&answer.NameShown); err != nil {
		return nil, fmt.Errorf("failed to read whether the player keeps out of the clips: %w", err)
	}
	return answer, nil
}

// Each link's address, where publishing has one; else empty, and the game links to the channel.
func (q *PostgresQuery) withAddresses(ctx context.Context, answer *clipsv1.GetAppearancesResponse) error {
	if len(answer.GetAppearances()) == 0 {
		return nil
	}
	posts := make([]string, 0, len(answer.GetAppearances()))
	for _, appearance := range answer.GetAppearances() {
		posts = append(posts, appearance.GetPostId())
	}
	addresses, err := q.links.Of(ctx, posts)
	if err != nil {
		return fmt.Errorf("failed to read where the posts are: %w", err)
	}
	for _, appearance := range answer.GetAppearances() {
		for _, link := range appearance.GetLinks() {
			link.Url = addresses[PostOnPlatform{Post: appearance.GetPostId(), Platform: link.GetPlatform()}]
		}
	}
	return nil
}
