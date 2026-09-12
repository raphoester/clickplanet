package bonus_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/bonus_click"
)

type stubClick struct{ err error }

func (s stubClick) Execute(context.Context, click.In) (click.Out, error) {
	return click.Out{}, s.err
}

type recordingPresence struct{ scopes []string }

func (r *recordingPresence) Clicked(scope string) { r.scopes = append(r.scopes, scope) }

func TestAnAcceptedClickMarksTheCallerAsPlaying(t *testing.T) {
	presence := &recordingPresence{}

	_, err := bonus_click.New(stubClick{}, presence).Execute(t.Context(), click.In{})
	require.NoError(t, err)

	assert.Len(t, presence.scopes, 1)
}

func TestARefusedClickIsNotPlaying(t *testing.T) {
	presence := &recordingPresence{}

	_, err := bonus_click.New(stubClick{err: errors.New("nope")}, presence).Execute(t.Context(), click.In{})

	require.Error(t, err)
	assert.Empty(t, presence.scopes, "a click that changed nothing is not playing")
}
