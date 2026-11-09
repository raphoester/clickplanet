package posts

import (
	"errors"
	"fmt"
	"strings"
)

// The kinds of file a job posts: a video, or a photo of a TikTok carousel.
const (
	VideoMP4  = "video/mp4"
	PhotoJPEG = "image/jpeg"
)

const maxFileName = 255

// The most bytes of each kind: a 22s clip is about 80 MB, and TikTok takes a photo of 20 MB at most.
var maxBytes = map[string]int64{VideoMP4: 1 << 30, PhotoJPEG: 20 << 20}

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
	most, known := maxBytes[m.ContentType]
	if !known {
		reasons = append(reasons, fmt.Errorf("the file is %q, and only %s and %s are posted", m.ContentType, VideoMP4, PhotoJPEG))
	}
	if known && (m.Size <= 0 || m.Size > most) {
		reasons = append(reasons, fmt.Errorf("the file is %d bytes, and a %s must be 1 to %d", m.Size, m.ContentType, most))
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
