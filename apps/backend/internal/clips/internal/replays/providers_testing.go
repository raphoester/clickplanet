//go:build testing

package replays

import "context"

type ScriptedSource struct {
	Asked  []Window
	replay Replay
	err    error

	RacesAsked int
	race       Race
	raceErr    error
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

var _ RaceSource = (*ScriptedSource)(nil)

func (s *ScriptedSource) AnswerRace(race Race) {
	s.race = race
}

func (s *ScriptedSource) FailRaceWith(err error) {
	s.raceErr = err
}

func (s *ScriptedSource) Race(context.Context) (Race, error) {
	s.RacesAsked++
	return s.race, s.raceErr
}

type MemoryStore struct {
	Saved      []Replay
	SavedRaces []Race
	err        error
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

var _ RaceStore = (*MemoryStore)(nil)

func (s *MemoryStore) SaveRace(_ context.Context, race Race) error {
	if s.err != nil {
		return s.err
	}
	s.SavedRaces = append(s.SavedRaces, race)
	return nil
}
