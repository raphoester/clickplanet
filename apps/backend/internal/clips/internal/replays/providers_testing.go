//go:build testing

package replays

import "context"

type ScriptedSource struct {
	Asked  []Window
	replay Replay
	err    error

	TurnsAsked int
	turn       Turn
	turnErr    error

	RoundsAsked []RoundID
	round       Round
	roundErr    error
}

func NewScriptedSource(replay Replay) *ScriptedSource {
	return &ScriptedSource{replay: replay}
}

var _ Source = (*ScriptedSource)(nil)

func (s *ScriptedSource) FailWith(err error) {
	s.err = err
}

func (s *ScriptedSource) Replay(_ context.Context, window Window) (Replay, error) {
	s.Asked = append(s.Asked, window)
	return s.replay, s.err
}

var _ TurnSource = (*ScriptedSource)(nil)

func (s *ScriptedSource) AnswerTurn(turn Turn) {
	s.turn = turn
}

func (s *ScriptedSource) FailTurnWith(err error) {
	s.turnErr = err
}

func (s *ScriptedSource) Turn(context.Context) (Turn, error) {
	s.TurnsAsked++
	return s.turn, s.turnErr
}

var _ RoundSource = (*ScriptedSource)(nil)

func (s *ScriptedSource) AnswerRound(round Round) {
	s.round = round
}

func (s *ScriptedSource) FailRoundWith(err error) {
	s.roundErr = err
}

func (s *ScriptedSource) Round(_ context.Context, round RoundID) (Round, error) {
	s.RoundsAsked = append(s.RoundsAsked, round)
	return s.round, s.roundErr
}

type MemoryStore struct {
	Saved       []Replay
	SavedTurns  []Turn
	SavedRounds []Round
	err         error
}

var _ Store = (*MemoryStore)(nil)

func (s *MemoryStore) FailWith(err error) {
	s.err = err
}

func (s *MemoryStore) Save(_ context.Context, replay Replay) error {
	if s.err != nil {
		return s.err
	}
	s.Saved = append(s.Saved, replay)
	return nil
}

var _ TurnStore = (*MemoryStore)(nil)

func (s *MemoryStore) SaveTurn(_ context.Context, turn Turn) error {
	if s.err != nil {
		return s.err
	}
	s.SavedTurns = append(s.SavedTurns, turn)
	return nil
}

var _ RoundStore = (*MemoryStore)(nil)

func (s *MemoryStore) SaveRound(_ context.Context, round Round) error {
	if s.err != nil {
		return s.err
	}
	s.SavedRounds = append(s.SavedRounds, round)
	return nil
}
