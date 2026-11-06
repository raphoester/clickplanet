package get_mission_totals_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_mission_totals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_mission_totals_handler/totals_query"
)

type window struct{ since, until time.Time }

type stubQuery struct {
	asked []window
	err   error
}

func (s *stubQuery) Totals(_ context.Context, since, until time.Time) (*missionsv1.GetMissionTotalsResponse, error) {
	s.asked = append(s.asked, window{since: since, until: until})
	return &missionsv1.GetMissionTotalsResponse{}, s.err
}

var since = time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)

func TestTheOperatorReadsTheTotalsOfAWindow(t *testing.T) {
	query := &stubQuery{}

	_, err := get_mission_totals_handler.New(query).GetMissionTotals(t.Context(),
		connect.NewRequest(&missionsv1.GetMissionTotalsRequest{
			Since: timestamppb.New(since), Until: timestamppb.New(since.Add(48 * time.Hour)),
		}))

	require.NoError(t, err)
	assert.Equal(t, []window{{since: since, until: since.Add(48 * time.Hour)}}, query.asked)
}

func TestAnUnsetUntilIsLeftToTheQuery(t *testing.T) {
	query := &stubQuery{}

	_, err := get_mission_totals_handler.New(query).GetMissionTotals(t.Context(),
		connect.NewRequest(&missionsv1.GetMissionTotalsRequest{Since: timestamppb.New(since)}))

	require.NoError(t, err)
	assert.Equal(t, []window{{since: since}}, query.asked)
}

func TestAWindowTheQueryRefusesIsTheCallersFault(t *testing.T) {
	query := &stubQuery{err: totals_query.ErrInvalidWindow}

	_, err := get_mission_totals_handler.New(query).GetMissionTotals(t.Context(),
		connect.NewRequest(&missionsv1.GetMissionTotalsRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
