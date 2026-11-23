//go:build testing

package inmemory_conversation_storage

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu    sync.Mutex
	notes []notes.NoteID
	kept  []conversations.Snapshot
	err   error
}

var _ conversations.Storage = (*Storage)(nil)

// A conversation about a note it was not told of is refused, as postgres refuses one.
func (s *Storage) KeepNote(note notes.NoteID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.notes = append(s.notes, note)
}

func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func (s *Storage) Seed(snapshots ...conversations.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.kept = append(s.kept, snapshots...)
}

func (s *Storage) Kept() []conversations.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.kept)
}

func (s *Storage) Save(_ context.Context, saved ...conversations.Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	for _, conversation := range saved {
		snapshot := conversation.TakeSnapshot()
		if !empty(snapshot) && !slices.Contains(s.notes, snapshot.Note) {
			return fmt.Errorf("%w: %s", notes.ErrUnknownNote, snapshot.Note)
		}
	}
	for _, conversation := range saved {
		snapshot := conversation.TakeSnapshot()
		s.kept = slices.DeleteFunc(s.kept, func(kept conversations.Snapshot) bool { return kept.Note == snapshot.Note })
		s.kept = append(s.kept, snapshot)
	}
	return nil
}

func (s *Storage) Conversations(_ context.Context, asked ...notes.NoteID) ([]conversations.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	found := make([]conversations.Conversation, 0, len(asked))
	for _, note := range asked {
		snapshot := conversations.Snapshot{Note: note}
		if at := slices.IndexFunc(s.kept, func(kept conversations.Snapshot) bool { return kept.Note == note }); at >= 0 {
			snapshot = s.kept[at]
		}
		conversation, err := snapshot.Restore()
		if err != nil {
			return nil, err //nolint:wrapcheck // the test reads the rule's own error.
		}
		found = append(found, conversation)
	}
	return found, nil
}

// Nothing to write: postgres asks no note for it.
func empty(snapshot conversations.Snapshot) bool {
	return len(snapshot.Answers) == 0 && len(snapshot.Replies) == 0 && snapshot.ClosedAt.IsZero()
}
