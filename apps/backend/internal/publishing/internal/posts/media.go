package posts

import (
	"errors"
	"fmt"
	"strings"
)

// The one kind of file a job posts.
const VideoMP4 = "video/mp4"

const (
	maxFileName = 255
	// A 22s clip is about 80 MB.
	maxMediaBytes = 1 << 30
)

var ErrInvalidMedia = errors.New("invalid media")

// A file a job is about to upload to the service's storage.
type Media struct {
	FileName    string
	ContentType string
	Size        int64
}

// Every reason the file cannot be posted, at once, as ErrInvalidMedia; nil when it can.
func (m Media) FileError() error {
	var reasons []error
	if m.FileName == "" || len(m.FileName) > maxFileName || strings.ContainsAny(m.FileName, `/\`) {
		reasons = append(reasons, fmt.Errorf("the file's name is empty, longer than %d or a path", maxFileName))
	}
	if m.ContentType != VideoMP4 {
		reasons = append(reasons, fmt.Errorf("the file is %q, and only %s is posted", m.ContentType, VideoMP4))
	}
	if m.Size <= 0 || m.Size > maxMediaBytes {
		reasons = append(reasons, fmt.Errorf("the file is %d bytes, and must be 1 to %d", m.Size, maxMediaBytes))
	}
	if len(reasons) > 0 {
		return fmt.Errorf("%w: %w", ErrInvalidMedia, errors.Join(reasons...))
	}
	return nil
}

// Where to upload a file, and the address a post then names it by.
type MediaUpload struct {
	// A credential until it expires: never logged.
	UploadURL string
	Media     MediaURL
}
