package videos_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

type Links interface {
	ViewURL(id videos.VideoID) (string, error)
	DownloadURL(id videos.VideoID, fileName string) (string, error)
}

const limit = 200

func NewPostgresQuery(db cppg.Conn, links Links) *PostgresQuery {
	return &PostgresQuery{db: db, links: links}
}

type PostgresQuery struct {
	db    cppg.Conn
	links Links
}

const listed = `
	SELECT video_id, name, caption, media_type, size_bytes, kept_at
	FROM videos
	ORDER BY kept_at DESC, video_id
	LIMIT $1
`

func (q *PostgresQuery) Videos(ctx context.Context) (*clipsv1.ListVideosResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, listed, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read the videos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := &clipsv1.ListVideosResponse{}
	for rows.Next() {
		var (
			id         uuid.UUID
			name       string
			caption    string
			media      string
			size       int64
			keptAt     time.Time
			videoID    videos.VideoID
			viewURL    string
			downloadTo string
		)
		if err := rows.Scan(&id, &name, &caption, &media, &size, &keptAt); err != nil {
			return nil, fmt.Errorf("failed to read a video: %w", err)
		}
		videoID = videos.VideoID(id)
		if viewURL, err = q.links.ViewURL(videoID); err != nil {
			return nil, fmt.Errorf("failed to link the video %s: %w", videoID, err)
		}
		if downloadTo, err = q.links.DownloadURL(videoID, videos.FileName(name, videos.MediaType(media))); err != nil {
			return nil, fmt.Errorf("failed to link the video %s: %w", videoID, err)
		}
		out.Videos = append(out.Videos, &clipsv1.Video{
			VideoId: videoID.String(), Name: name, Caption: caption, ContentType: media,
			SizeBytes: uint64(size), KeptAt: timestamppb.New(keptAt), //nolint:gosec // the CHECK keeps it positive.
			ViewUrl: viewURL, DownloadUrl: downloadTo,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the videos: %w", err)
	}
	return out, nil
}
