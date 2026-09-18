package bonus_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/bonus_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubClick struct{ err error }

func (s stubClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return click_usecase.Out{}, s.err
}

type recordingPresence struct {
	scopes  []string
	holders []bonuses.Holder
}

func (r *recordingPresence) Clicked(scope string, holder bonuses.Holder) {
	r.scopes = append(r.scopes, scope)
	r.holders = append(r.holders, holder)
}

func TestAnAcceptedClickMarksTheCallerAsPlaying(t *testing.T) {
	presence := &recordingPresence{}

	_, err := bonus_click.New(stubClick{}, presence).Execute(t.Context(), click_usecase.In{})
	require.NoError(t, err)

	assert.Len(t, presence.scopes, 1)
}

func TestARefusedClickIsNotPlaying(t *testing.T) {
	presence := &recordingPresence{}

	_, err := bonus_click.New(stubClick{err: errors.New("nope")}, presence).Execute(t.Context(), click_usecase.In{})

	require.Error(t, err)
	assert.Empty(t, presence.scopes, "a click that changed nothing is not playing")
}

func TestAClickSaysWhichAccountPlaysBehindTheScope(t *testing.T) {
	presence := &recordingPresence{}
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	_, err := bonus_click.New(stubClick{}, presence).Execute(ctx, click_usecase.In{})
	require.NoError(t, err)

	assert.Equal(t, []string{"1.2.3.4"}, presence.scopes)
	assert.Equal(t, []bonuses.Holder{"a-guest"}, presence.holders)
}
