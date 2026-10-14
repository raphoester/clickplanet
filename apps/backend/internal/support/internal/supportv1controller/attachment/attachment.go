package attachment

import (
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func IDs(sent []string) ([]notes.AttachmentID, error) {
	if len(sent) == 0 {
		return nil, nil
	}

	ids := make([]notes.AttachmentID, 0, len(sent))
	for _, value := range sent {
		id, err := notes.AttachmentIDOf(value)
		if err != nil {
			return nil, err //nolint:wrapcheck // the rule named it.
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func Uploaded(id notes.AttachmentID, link string) *supportv1.CreateUploadResponse {
	return &supportv1.CreateUploadResponse{AttachmentId: id.String(), Url: link}
}
