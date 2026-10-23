package find_authors_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/find_authors_handler"
)

var (
	eire  = players.AccountID{15: 1}
	eire1 = players.AccountID{15: 2}
)

type find struct {
	folded string
	limit  int
}

type stubNames struct {
	found []players.AccountID
	total int
	err   error
	asked []find
}

func (s *stubNames) Find(_ context.Context, folded string, limit int) ([]players.AccountID, int, error) {
	s.asked = append(s.asked, find{folded: folded, limit: limit})
	return s.found, s.total, s.err
}

type stubAuthors struct {
	answer *playerv1.GetAuthorsResponse
	err    error
	asked  [][]players.AccountID
}

func (s *stubAuthors) Authors(_ context.Context, accounts []players.AccountID) (*playerv1.GetAuthorsResponse, error) {
	s.asked = append(s.asked, accounts)
	return s.answer, s.err
}

func found() (*stubNames, *stubAuthors) {
	return &stubNames{found: []players.AccountID{eire, eire1}, total: 3},
		&stubAuthors{answer: &playerv1.GetAuthorsResponse{Authors: []*playerv1.Author{
			{AccountId: eire.String(), Name: "EIRE"},
			{AccountId: eire1.String(), Name: "EIRE1"},
		}}}
}

func findAuthors(
	t *testing.T, names *stubNames, authors *stubAuthors, req *playerv1.FindAuthorsRequest,
) (*connect.Response[playerv1.FindAuthorsResponse], error) {
	t.Helper()
	return find_authors_handler.New(names, authors).FindAuthors(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheNameIsFoldedAndEachAccountFoundIsNamedAsGetAuthorsNamesIt(t *testing.T) {
	names, authors := found()

	res, err := findAuthors(t, names, authors, &playerv1.FindAuthorsRequest{Name: "  Éire "})
	require.NoError(t, err)

	assert.Equal(t, []find{{folded: "éire", limit: 20}}, names.asked)
	assert.Equal(t, [][]players.AccountID{{eire, eire1}}, authors.asked)
	assert.True(t, proto.Equal(&playerv1.FindAuthorsResponse{Authors: authors.answer.GetAuthors(), Total: 3}, res.Msg))
}

func TestAnEmptyNameIsRefusedAndReadsNothing(t *testing.T) {
	for _, name := range []string{"", "   "} {
		names, authors := found()
		_, err := findAuthors(t, names, authors, &playerv1.FindAuthorsRequest{Name: name})
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), name)
		assert.Empty(t, names.asked, name)
		assert.Empty(t, authors.asked, name)
	}
}

func TestTheLimitAskedIsCapped(t *testing.T) {
	for asked, want := range map[uint32]int{7: 7, 500: 500, 10_000: 500} {
		names, authors := found()
		_, err := findAuthors(t, names, authors, &playerv1.FindAuthorsRequest{Name: "eire", Limit: asked})
		require.NoError(t, err)
		assert.Equal(t, []find{{folded: "eire", limit: want}}, names.asked, asked)
	}
}

func TestTheAnswerIsNeverStored(t *testing.T) {
	names, authors := found()
	res, err := findAuthors(t, names, authors, &playerv1.FindAuthorsRequest{Name: "eire"})
	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "who a player is changes when it renames")
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	names, authors := found()
	names.err = failure
	_, err := findAuthors(t, names, authors, &playerv1.FindAuthorsRequest{Name: "eire"})
	require.ErrorIs(t, err, failure)
	assert.Empty(t, authors.asked)

	names, authors = found()
	authors.err = failure
	_, err = findAuthors(t, names, authors, &playerv1.FindAuthorsRequest{Name: "eire"})
	assert.ErrorIs(t, err, failure)
}
