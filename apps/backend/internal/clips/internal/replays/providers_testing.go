//go:build testing

package replays

import "context"

type ScriptedSource struct {
	Asked  []Window
	replay Replay

	TurnsAsked int
	turn       Turn

	RoundsAsked []RoundID
	round       Round

	CastsAsked []Window
	cast       Cast

	HistoriesAsked []Window
	history        History

	FlagListsAsked int
	flagList       FlagList
}

func NewScriptedSource(replay Replay) *ScriptedSource {
	return &ScriptedSource{replay: replay}
}

var _ Source = (*ScriptedSource)(nil)

func (s *ScriptedSource) Replay(_ context.Context, window Window) (Replay, error) {
	s.Asked = append(s.Asked, window)
	return s.replay, nil
}

var _ TurnSource = (*ScriptedSource)(nil)

func (s *ScriptedSource) AnswerTurn(turn Turn) {
	s.turn = turn
}

func (s *ScriptedSource) Turn(context.Context) (Turn, error) {
	s.TurnsAsked++
	return s.turn, nil
}

var _ RoundSource = (*ScriptedSource)(nil)

func (s *ScriptedSource) AnswerRound(round Round) {
	s.round = round
}

func (s *ScriptedSource) Round(_ context.Context, round RoundID) (Round, error) {
	s.RoundsAsked = append(s.RoundsAsked, round)
	return s.round, nil
}

type MemoryStore struct {
	Saved          []Replay
	SavedTurns     []Turn
	SavedRounds    []Round
	SavedHistories []History
}

var _ Store = (*MemoryStore)(nil)

func (s *MemoryStore) Save(_ context.Context, replay Replay) error {
	s.Saved = append(s.Saved, replay)
	return nil
}

var _ TurnStore = (*MemoryStore)(nil)

func (s *MemoryStore) SaveTurn(_ context.Context, turn Turn) error {
	s.SavedTurns = append(s.SavedTurns, turn)
	return nil
}

var _ RoundStore = (*MemoryStore)(nil)

func (s *MemoryStore) SaveRound(_ context.Context, round Round) error {
	s.SavedRounds = append(s.SavedRounds, round)
	return nil
}

var _ CastSource = (*ScriptedSource)(nil)

func (s *ScriptedSource) AnswerCast(cast Cast) {
	s.cast = cast
}

func (s *ScriptedSource) Cast(_ context.Context, window Window) (Cast, error) {
	s.CastsAsked = append(s.CastsAsked, window)
	return s.cast, nil
}

var _ HistorySource = (*ScriptedSource)(nil)

func (s *ScriptedSource) AnswerHistory(history History) {
	s.history = history
}

func (s *ScriptedSource) History(_ context.Context, window Window) (History, error) {
	s.HistoriesAsked = append(s.HistoriesAsked, window)
	return s.history, nil
}

var _ HistoryStore = (*MemoryStore)(nil)

func (s *MemoryStore) SaveHistory(_ context.Context, history History) error {
	s.SavedHistories = append(s.SavedHistories, history)
	return nil
}

var _ FlagListSource = (*ScriptedSource)(nil)

func (s *ScriptedSource) AnswerFlagList(list FlagList) {
	s.flagList = list
}

func (s *ScriptedSource) FlagList(context.Context) (FlagList, error) {
	s.FlagListsAsked++
	return s.flagList, nil
}
