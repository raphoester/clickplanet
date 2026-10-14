//go:build testing

package inmemory_attachment_bucket

import (
	"context"
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New() *Bucket {
	return &Bucket{files: map[notes.AttachmentID]notes.Attachment{}}
}

type Bucket struct {
	mu    sync.Mutex
	files map[notes.AttachmentID]notes.Attachment
	err   error
}

func (b *Bucket) FailWith(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.err = err
}

// What a page uploaded through the links.
func (b *Bucket) Put(files ...notes.Attachment) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, file := range files {
		b.files[file.ID] = file
	}
}

func URL(id notes.AttachmentID) string {
	return fmt.Sprintf("memory://%s", id)
}

func (b *Bucket) UploadURL(file notes.Attachment) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return "", b.err
	}
	return URL(file.ID), nil
}

func (b *Bucket) Attachments(_ context.Context, ids ...notes.AttachmentID) ([]notes.Attachment, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}
	files := make([]notes.Attachment, 0, len(ids))
	for _, id := range ids {
		file, kept := b.files[id]
		if !kept {
			return nil, fmt.Errorf("%w: %s", notes.ErrNotUploaded, id)
		}
		files = append(files, file)
	}
	return files, nil
}

func (b *Bucket) Delete(_ context.Context, ids ...notes.AttachmentID) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}
	for _, id := range ids {
		delete(b.files, id)
	}
	return nil
}
