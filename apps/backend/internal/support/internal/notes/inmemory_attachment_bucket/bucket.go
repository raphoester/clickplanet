//go:build testing

package inmemory_attachment_bucket

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New() *Bucket {
	return &Bucket{}
}

type Bucket struct {
	mu      sync.Mutex
	deleted []string
	err     error
}

func (b *Bucket) FailWith(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.err = err
}

func URL(note notes.NoteID, position int) string {
	return fmt.Sprintf("memory://%s/%d", uuid.UUID(note), position)
}

func (b *Bucket) UploadURLs(note notes.Note) ([]string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}
	return urls(note), nil
}

func (b *Bucket) Delete(_ context.Context, of ...notes.Note) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}
	for _, note := range of {
		b.deleted = append(b.deleted, urls(note)...)
	}
	return nil
}

// Every file a Delete was asked for, in the order asked.
func (b *Bucket) Deleted() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]string(nil), b.deleted...)
}

func urls(note notes.Note) []string {
	kept := note.TakeSnapshot()

	files := make([]string, 0, len(kept.Attachments))
	for position := range kept.Attachments {
		files = append(files, URL(kept.ID, position))
	}
	return files
}
