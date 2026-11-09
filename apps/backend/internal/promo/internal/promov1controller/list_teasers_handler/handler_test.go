package list_teasers_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_teasers_handler"
)

type query struct {
	asked []time.Duration
}

func (q *query) Teasers(_ context.Context, within time.Duration) ([]*promov1.Teaser, error) {
	q.asked = append(q.asked, within)
	return []*promov1.Teaser{{BattleId: "ring-war"}}, nil
}

func TestTheTeasersOfAWindowOfUpToAWeekAreRead(t *testing.T) {
	read := &query{}

	answer, err := list_teasers_handler.New(read).ListTeasers(t.Context(),
		connect.NewRequest(&promov1.ListTeasersRequest{Within: durationpb.New(36 * time.Hour)}))

	require.NoError(t, err)
	assert.Equal(t, "ring-war", answer.Msg.GetTeasers()[0].GetBattleId())
	assert.Equal(t, []time.Duration{36 * time.Hour}, read.asked)
}

func TestAWindowOfNothingOrOfMoreThanAWeekIsInvalidArgument(t *testing.T) {
	for _, within := range []*durationpb.Duration{nil, durationpb.New(-time.Hour), durationpb.New(8 * 24 * time.Hour)} {
		read := &query{}

		_, err := list_teasers_handler.New(read).ListTeasers(t.Context(), connect.NewRequest(&promov1.ListTeasersRequest{Within: within}))

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", within)
		assert.Empty(t, read.asked)
	}
}
