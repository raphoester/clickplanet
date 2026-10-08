package get_takes_by_country_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *planetv1.GetTakesByCountryResponse
	err    error
	asked  []ledger.AccountID
}

func (s *stubQuery) TakesByCountry(_ context.Context, account ledger.AccountID) (*planetv1.GetTakesByCountryResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func ask(t *testing.T, query *stubQuery, account string) (*connect.Response[planetv1.GetTakesByCountryResponse], error) {
	t.Helper()

	return get_takes_by_country_handler.New(query).GetTakesByCountry(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&planetv1.GetTakesByCountryRequest{AccountId: account}))
}

func TestTheAccountsAnswerIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetTakesByCountryResponse{TakenFor: []*planetv1.CountryTakes{{CountryId: "fr", Tiles: 4}}}}

	res, err := ask(t, query, ada)

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	account, err := ledger.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []ledger.AccountID{account}, query.asked)
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
