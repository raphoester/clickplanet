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

type TurnStore string

var _ replays.TurnStore = TurnStore("")

func (s TurnStore) SaveTurn(_ context.Context, turn replays.Turn) error {
	if err := os.WriteFile(string(s), turn.Document, 0o600); err != nil {
		return fmt.Errorf("failed to write the turn: %w", err)
	}
	return nil
}

type RoundStore string

var _ replays.RoundStore = RoundStore("")

func (s RoundStore) SaveRound(_ context.Context, round replays.Round) error {
	if err := os.WriteFile(string(s), round.Document, 0o600); err != nil {
		return fmt.Errorf("failed to write the round: %w", err)
	}
	return nil
}

type CastStore string

var _ replays.CastStore = CastStore("")

func (s CastStore) SaveCast(_ context.Context, cast replays.Cast) error {
	if err := os.WriteFile(string(s), cast.Document, 0o600); err != nil {
		return fmt.Errorf("failed to write the cast: %w", err)
	}
	return nil
}
