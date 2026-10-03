package notifying_award_titles_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase/notifying_award_titles"
)

var ada = players.AccountID{15: 1}

type stubUseCase struct {
	granted titles.IDs
	err     error
}

func (s stubUseCase) Execute(context.Context, players.AccountID) (titles.IDs, error) {
	return s.granted, s.err
}

type recordingFeed struct {
	published []titles.IDs
}

func (r *recordingFeed) Publish(account players.AccountID, earned titles.IDs) {
	if account == ada {
		r.published = append(r.published, earned)
	}
}

func TestWhatATakeGrantedReachesTheAccountsStreams(t *testing.T) {
	feed := &recordingFeed{}

	granted, err := notifying_award_titles.New(stubUseCase{granted: titles.IDs{"warlord"}}, feed).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"warlord"}, granted)
	assert.Equal(t, []titles.IDs{{"warlord"}}, feed.published)
}

func TestNothingGrantedOrAFailureNotifiesNobody(t *testing.T) {
	feed := &recordingFeed{}
	cause := errors.New("postgres is down")

	_, err := notifying_award_titles.New(stubUseCase{}, feed).Execute(t.Context(), ada)
	require.NoError(t, err)
	_, err = notifying_award_titles.New(stubUseCase{granted: titles.IDs{"warlord"}, err: cause}, feed).Execute(t.Context(), ada)
	require.ErrorIs(t, err, cause)

	assert.Empty(t, feed.published)
}
