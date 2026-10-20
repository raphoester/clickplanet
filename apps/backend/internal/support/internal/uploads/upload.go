package uploads

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	ErrInvalidUpload = errors.New("the upload is invalid")
	ErrShelfFull     = errors.New("the bucket holds as many bytes as it may")
)

// A file a page was given a link to put in the bucket. It takes its room from the moment the link is signed,
// whether the file comes or not, for a link cannot be taken back.
type Upload struct {
	id        notes.AttachmentID
	size      int64
	at        time.Time
	discarded bool
}

func NewUpload(file notes.Attachment, at time.Time) (Upload, error) {
	return Snapshot{ID: file.ID, Size: file.Size, At: at}.Restore()
}

// A file no note named in time: its file is deleted, its storage keeps it no more, and its room is free again.
func (u Upload) Discarded() Upload {
	u.discarded = true
	return u
}

func AllDiscarded(unsent []Upload) []Upload {
	discarded := make([]Upload, 0, len(unsent))
	for _, upload := range unsent {
		discarded = append(discarded, upload.Discarded())
	}
	return discarded
}

// The files of the uploads, in their order.
func FilesOf(kept []Upload) []notes.AttachmentID {
	ids := make([]notes.AttachmentID, 0, len(kept))
	for _, upload := range kept {
		ids = append(ids, upload.id)
	}
	return ids
}

func (u Upload) TakeSnapshot() Snapshot {
	return Snapshot{ID: u.id, Size: u.size, At: u.at, Discarded: u.discarded}
}

// The state of an Upload, for a store. It holds no rule: Restore does.
type Snapshot struct {
	ID        notes.AttachmentID
	Size      int64
	At        time.Time
	Discarded bool
}

func (s Snapshot) Restore() (Upload, error) {
	switch {
	case s.ID == notes.AttachmentID{}:
		return Upload{}, fmt.Errorf("%w: no id", ErrInvalidUpload)
	case s.Size <= 0:
		return Upload{}, fmt.Errorf("%w: no size", ErrInvalidUpload)
	case s.At.IsZero():
		return Upload{}, fmt.Errorf("%w: no time", ErrInvalidUpload)
	}
	return Upload{id: s.ID, size: s.Size, at: s.At, discarded: s.Discarded}, nil
}

// A storage finds uploads and saves them. It holds no rule: what changes an upload is a method of Upload.
type Storage interface {
	// The bytes of every upload kept.
	Held(ctx context.Context) (int64, error)
	// Under one lock for every upload: the bytes held, then the upload kept unless check refuses it, with check's
	// error. Two reservations never read the same bytes held.
	Reserve(ctx context.Context, upload Upload, check func(held int64) error) error
	// The uploads kept from before the cutoff that no note names: oldest first, limit at most.
	Unsent(ctx context.Context, cutoff time.Time, limit int) ([]Upload, error)
	// An upload saved again replaces itself, and a discarded one is kept no more. All of them, or none.
	Save(ctx context.Context, uploads ...Upload) error
}

// How many bytes the bucket may hold: the files notes carry, and the files that may still come.
type Shelf struct {
	storage  Storage
	capacity int64
	clock    cptime.Clock
}

func NewShelf(storage Storage, capacity int64, clock cptime.Clock) *Shelf {
	return &Shelf{storage: storage, capacity: capacity, clock: clock}
}

// Keeps the file's room, or refuses it with ErrShelfFull.
func (s *Shelf) Reserve(ctx context.Context, file notes.Attachment) error {
	upload, err := NewUpload(file, s.clock.Now())
	if err != nil {
		return err
	}

	err = s.storage.Reserve(ctx, upload, func(held int64) error {
		if held+file.Size > s.capacity {
			return fmt.Errorf("%w: %d bytes held, %d asked, and it holds %d at most", ErrShelfFull, held, file.Size, s.capacity)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to keep room for the file: %w", err)
	}
	return nil
}

// The bytes the bucket may still take.
func (s *Shelf) Room(ctx context.Context) (int64, error) {
	held, err := s.storage.Held(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to read the bytes the bucket holds: %w", err)
	}
	return max(s.capacity-held, 0), nil
}
