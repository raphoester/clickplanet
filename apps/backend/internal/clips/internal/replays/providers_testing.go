//go:build testing

package replays

import "context"

type ScriptedSource struct {
	Asked  []Window
	replay Replay
	err    error
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

type MemoryStore struct {
	Saved []Replay
	err   error
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
