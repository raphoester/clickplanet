package links_query

import (
	"context"
	"fmt"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// The newest address a snapshot read of each post on each platform: the daily snapshot is the one read of Zernio that
// keeps it.
const links = `
	SELECT DISTINCT ON (post_id, platform) post_id, platform, document->>'url'
	FROM snapshot_rows
	WHERE post_id = ANY($1) AND COALESCE(document->>'url', '') <> ''
	ORDER BY post_id, platform, taken_at DESC
`

func (q *PostgresQuery) Links(ctx context.Context, posts []string) (*publishingv1.GetPostLinksResponse, error) {
	answer := &publishingv1.GetPostLinksResponse{}
	if len(posts) == 0 {
		return answer, nil
	}
	rows, err := q.db.On(ctx).QueryContext(ctx, links, posts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the posts' links: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		link := &publishingv1.PostLink{}
		if err := rows.Scan(&link.PostId, &link.Platform, &link.Url); err != nil {
			return nil, fmt.Errorf("failed to read a post's link: %w", err)
		}
		answer.Links = append(answer.Links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the posts' links: %w", err)
	}
	return answer, nil
}
