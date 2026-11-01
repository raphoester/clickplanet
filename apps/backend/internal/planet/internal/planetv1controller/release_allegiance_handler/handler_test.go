package release_allegiance_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/release_allegiance_handler"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubBook struct {
	err      error
	released []allegiances.Account
}

func (s *stubBook) Release(_ context.Context, account allegiances.Account) error {
	s.released = append(s.released, account)
	return s.err
}

func release(
	t *testing.T, book *stubBook, lock allegiances.Lock, account string,
) (*connect.Response[planetv1.ReleaseAllegianceResponse], error) {
	t.Helper()

	handler := release_allegiance_handler.New(book, func() allegiances.Lock { return lock })
	req := connect.NewRequest(&planetv1.ReleaseAllegianceRequest{AccountId: account})
	return handler.ReleaseAllegiance(t.Context(), req) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheAccountIsReleased(t *testing.T) {
	book := &stubBook{}

	res, err := release(t, book, false, ada)

	require.NoError(t, err)
	assert.Equal(t, []allegiances.Account{ada}, book.released)
	assert.False(t, res.Msg.GetLocked())
}

func TestTheAnswerSaysWhenTheRulesLockEveryFlag(t *testing.T) {
	res, err := release(t, &stubBook{}, true, ada)

	require.NoError(t, err)
	assert.True(t, res.Msg.GetLocked())
}

func TestAnIDThatIsNotAnAccountIsInvalidAndReleasesNothing(t *testing.T) {
	book := &stubBook{}

	for _, account := range []string{"", "nope", "00000000-0000-0000-0000-000000000000"} {
		_, err := release(t, book, false, account)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%q", account)
	}
	assert.Empty(t, book.released)
}

func TestAFailedReleaseIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := release(t, &stubBook{err: failure}, false, ada)

	assert.ErrorIs(t, err, failure)
}
