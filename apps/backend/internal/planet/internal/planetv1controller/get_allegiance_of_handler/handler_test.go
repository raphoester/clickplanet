package get_allegiance_of_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_of_handler"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *planetv1.GetAllegianceResponse
	err    error
	asked  []allegiances.Account
}

func (s *stubQuery) Allegiance(_ context.Context, account allegiances.Account) (*planetv1.GetAllegianceResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func ask(t *testing.T, query *stubQuery, account string) (*connect.Response[planetv1.GetAllegianceOfResponse], error) {
	t.Helper()

	return get_allegiance_of_handler.New(query).GetAllegianceOf(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&planetv1.GetAllegianceOfRequest{AccountId: account}))
}

func TestTheAccountsAllegianceIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetAllegianceResponse{Allegiance: &planetv1.Allegiance{CountryId: "fr"}}}

	res, err := ask(t, query, ada)

	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.Allegiance{CountryId: "fr"}, res.Msg.GetAllegiance()))
	assert.Equal(t, []allegiances.Account{ada}, query.asked)
}

func TestAnAccountWithNoFlagHasNoAllegiance(t *testing.T) {
	res, err := ask(t, &stubQuery{answer: &planetv1.GetAllegianceResponse{}}, ada)

	require.NoError(t, err)
	assert.Nil(t, res.Msg.GetAllegiance())
}

func TestAnIDThatIsNotAnAccountIsInvalidAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	for _, account := range []string{"", "nope", "00000000-0000-0000-0000-000000000000"} {
		_, err := ask(t, query, account)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%q", account)
	}
	assert.Empty(t, query.asked)
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := ask(t, &stubQuery{err: failure}, ada)

	assert.ErrorIs(t, err, failure)
}
