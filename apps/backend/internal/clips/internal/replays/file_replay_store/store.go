package file_replay_store

import (
	"context"
	"fmt"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

type Store string

var _ replays.Store = Store("")

func (s Store) Save(_ context.Context, replay replays.Replay) error {
	if err := os.WriteFile(string(s), replay.Document, 0o600); err != nil {
		return fmt.Errorf("failed to write the replay: %w", err)
	}
	return nil
}
