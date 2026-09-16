package unban_member_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/unban_member_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type fakeUnbanner struct {
	lifted []string
	err    error
}

func (f *fakeUnbanner) Unban(_ context.Context, tag string) error {
	if f.err != nil {
		return f.err
	}
	f.lifted = append(f.lifted, tag)
	return nil
}

func TestABanIsLiftedOnTheTagTheChatShows(t *testing.T) {
	unbanner := &fakeUnbanner{}

	out, err := unban_member_usecase.New(unbanner).
		Execute(t.Context(), unban_member_usecase.In{AuthorTag: "#A1B2C3"})

	require.NoError(t, err)
	assert.Equal(t, []string{"a1b2c3"}, unbanner.lifted)
	assert.Equal(t, "a1b2c3", out.AuthorTag)
}

func TestSomethingThatIsNotATagLiftsNothing(t *testing.T) {
	unbanner := &fakeUnbanner{}

	_, err := unban_member_usecase.New(unbanner).
		Execute(t.Context(), unban_member_usecase.In{AuthorTag: "Bob"})

	require.ErrorIs(t, err, messages.ErrInvalidTag)
	assert.Empty(t, unbanner.lifted)
}

func TestLiftingABanNobodyPassedIsReported(t *testing.T) {
	_, err := unban_member_usecase.New(&fakeUnbanner{err: bans.ErrNotBanned}).
		Execute(t.Context(), unban_member_usecase.In{AuthorTag: "a1b2c3"})

	require.ErrorIs(t, err, bans.ErrNotBanned)
}
