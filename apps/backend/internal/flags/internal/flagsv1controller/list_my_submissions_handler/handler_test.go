package list_my_submissions_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_my_submissions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type query struct {
	asked []submissions.AccountID
}

func (q *query) Mine(_ context.Context, account submissions.AccountID) (*flagsv1.ListMySubmissionsResponse, error) {
	q.asked = append(q.asked, account)
	return &flagsv1.ListMySubmissionsResponse{Submissions: []*flagsv1.Submission{{Id: "s"}}}, nil
}

func TestAGuestFoundedNothingAndIsNotRead(t *testing.T) {
	q := &query{}
	ctx := cpctx.AddAccountToContext(t.Context(), "00000000-0000-0000-0000-000000000001")

	res, err := list_my_submissions_handler.New(q).ListMySubmissions(ctx, connect.NewRequest(&flagsv1.ListMySubmissionsRequest{}))

	require.NoError(t, err)
	assert.Empty(t, res.Msg.GetSubmissions())
	assert.Empty(t, q.asked)
}

func TestASignedInAccountReadsItsOwn(t *testing.T) {
	q := &query{}
	ctx := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(t.Context(), "00000000-0000-0000-0000-000000000001"))

	res, err := list_my_submissions_handler.New(q).ListMySubmissions(ctx, connect.NewRequest(&flagsv1.ListMySubmissionsRequest{}))

	require.NoError(t, err)
	assert.Len(t, res.Msg.GetSubmissions(), 1)
	assert.Equal(t, []submissions.AccountID{{15: 1}}, q.asked)
}
