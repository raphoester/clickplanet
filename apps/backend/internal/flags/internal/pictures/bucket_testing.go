//go:build testing

package pictures

import (
	"context"
	"fmt"
	"maps"
	"sync"
)

var _ Bucket = (*MemoryBucket)(nil)

// MemoryBucket keeps the files uploaded and the pictures made public.
type MemoryBucket struct {
	mu        sync.Mutex
	uploads   map[UploadID][]byte
	published map[string][]byte
}

func NewMemoryBucket() *MemoryBucket {
	return &MemoryBucket{uploads: map[UploadID][]byte{}, published: map[string][]byte{}}
}

// Upload is what the page sends to the link.
func (m *MemoryBucket) Upload(id UploadID, file []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploads[id] = file
}

func (m *MemoryBucket) UploadURL(file File) (string, error) {
	return "https://bucket.test/" + file.ID.Key(), nil
}

func (m *MemoryBucket) Uploaded(_ context.Context, id UploadID, most int64) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	file, ok := m.uploads[id]
	switch {
	case !ok:
		return nil, fmt.Errorf("%w: %s", ErrUnknownUpload, id)
	case int64(len(file)) > most:
		return nil, ErrTooLarge
	}
	return file, nil
}

func (m *MemoryBucket) Publish(_ context.Context, drawn Drawn) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published[drawn.Picture.Key()] = drawn.PNG
	return nil
}

func (m *MemoryBucket) Discard(_ context.Context, id UploadID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.uploads, id)
	return nil
}

func (m *MemoryBucket) Published() map[string][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.published)
}

func (m *MemoryBucket) Uploads() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.uploads)
}
