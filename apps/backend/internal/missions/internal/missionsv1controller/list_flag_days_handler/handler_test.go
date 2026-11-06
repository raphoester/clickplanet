package list_flag_days_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/list_flag_days_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/list_flag_days_handler/flag_days_query"
)

type window struct{ since, until time.Time }

type stubQuery struct {
	asked []window
	err   error
}

func (s *stubQuery) FlagDays(_ context.Context, since, until time.Time) (*missionsv1.ListFlagDaysResponse, error) {
	s.asked = append(s.asked, window{since: since, until: until})
	return &missionsv1.ListFlagDaysResponse{}, s.err
}

var since = time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)

func TestTheOperatorReadsTheFlagDaysOfAWindow(t *testing.T) {
	query := &stubQuery{}

	_, err := list_flag_days_handler.New(query).ListFlagDays(t.Context(), connect.NewRequest(&missionsv1.ListFlagDaysRequest{
		Since: timestamppb.New(since), Until: timestamppb.New(since.Add(48 * time.Hour)),
	}))

	require.NoError(t, err)
	assert.Equal(t, []window{{since: since, until: since.Add(48 * time.Hour)}}, query.asked)
}

func TestAnUnsetUntilIsLeftToTheQuery(t *testing.T) {
	query := &stubQuery{}

	_, err := list_flag_days_handler.New(query).ListFlagDays(t.Context(),
		connect.NewRequest(&missionsv1.ListFlagDaysRequest{Since: timestamppb.New(since)}))

	require.NoError(t, err)
	assert.Equal(t, []window{{since: since}}, query.asked)
}

func TestAWindowTheQueryRefusesIsTheCallersFault(t *testing.T) {
	query := &stubQuery{err: flag_days_query.ErrInvalidWindow}

	_, err := list_flag_days_handler.New(query).ListFlagDays(t.Context(), connect.NewRequest(&missionsv1.ListFlagDaysRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
