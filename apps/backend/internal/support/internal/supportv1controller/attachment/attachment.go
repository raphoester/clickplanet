package attachment

import (
	"math"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func Declared(sent []*supportv1.Attachment) []notes.Attachment {
	if len(sent) == 0 {
		return nil
	}

	files := make([]notes.Attachment, 0, len(sent))
	for _, file := range sent {
		files = append(files, notes.Attachment{
			MediaType: notes.MediaType(file.GetContentType()),
			Size:      int64(min(file.GetSizeBytes(), math.MaxInt64)), //nolint:gosec // capped on this line.
		})
	}
	return files
}

func Uploads(links []string) []*supportv1.Upload {
	if len(links) == 0 {
		return nil
	}

	uploads := make([]*supportv1.Upload, 0, len(links))
	for _, link := range links {
		uploads = append(uploads, &supportv1.Upload{Url: link})
	}
	return uploads
}
