package flag_days_query_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/postgres_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/postgres_flag_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/list_flag_days_handler/flag_days_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db       *cppg.Postgres
	calendar *postgres_day_store.Store
	flags    *postgres_flag_day_store.Store
}

var (
	firstStart = time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)
	now        = firstStart.Add(72 * time.Hour)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "missions", migrations.FS)
	s.calendar = postgres_day_store.New(s.db)
	s.flags = postgres_flag_day_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) query() *flag_days_query.PostgresQuery {
	return flag_days_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now))
}

// The day told n days after the first.
func (s *testSuite) told(n int) days.Day {
	starts := firstStart.Add(time.Duration(n) * 24 * time.Hour)
	day, err := days.DayOf(days.DaySnapshot{
		ID: days.DayID(fmt.Sprintf("season-1/day-%d", n+1)), StartsAt: starts, EndsAt: starts.Add(24 * time.Hour),
	})
	s.Require().NoError(err)
	s.Require().NoError(s.calendar.KeepDay(s.T().Context(), day))
	return day
}

func (s *testSuite) played(day days.Day, flag string, snapshot flagmissions.DaySnapshot) {
	drawn, err := flagmissions.DayOf(flagmissions.DaySnapshot{Mission: snapshot.Mission})
	s.Require().NoError(err)
	_, err = s.flags.KeepDay(s.T().Context(), day, flag, drawn)
	s.Require().NoError(err)
	s.Require().NoError(s.flags.ChangeDay(s.T().Context(), day, flag, func(flagmissions.Day) flagmissions.Day {
		changed, err := flagmissions.DayOf(snapshot)
		s.Require().NoError(err)
		return changed
	}))
}

func (s *testSuite) protoOf(day days.Day) *missionsv1.Day {
	snapshot := day.Snapshot()
	return &missionsv1.Day{
		Id: string(snapshot.ID), StartsAt: timestamppb.New(snapshot.StartsAt), EndsAt: timestamppb.New(snapshot.EndsAt),
	}
}

func (s *testSuite) TestEveryKindTheStoreKeepsReadsAsItsPlayersSeeIt() {
	day := s.told(0)
	for n, status := range []flagmissions.Status{
		{
			Mission:  flagmissions.MissionSnapshot{ID: flagmissions.MissionID{15: 1}, Order: flagmissions.TakeIn{Ground: "ru", Goal: 400}, Points: 4},
			Progress: 120,
		},
		{
			Mission:  flagmissions.MissionSnapshot{ID: flagmissions.MissionID{15: 2}, Order: flagmissions.BreakShields{Flag: "se", Goal: 30}, Points: 9},
			Progress: 30, Done: true,
		},
	} {
		mission := status.Mission
		s.played(day, []string{"de", "fr"}[n], flagmissions.DaySnapshot{
			Mission: &mission, Progress: status.Progress, Done: status.Done, Takes: uint64(100 * (n + 1)), Strikes: uint64(n * 7),
		})
	}

	answer, err := s.query().FlagDays(s.T().Context(), firstStart, now)

	s.Require().NoError(err)
	flags := []string{"de", "fr"}
	want := make([]*missionsv1.FlagDay, 0, len(flags))
	for n, flag := range flags {
		read, err := s.flags.Day(s.T().Context(), day, flag)
		s.Require().NoError(err)
		status, ok := read.Status()
		s.Require().True(ok)
		mission, err := flagmessage.Mission(status)
		s.Require().NoError(err)
		want = append(want, &missionsv1.FlagDay{
			CountryId: flag, TilesTaken: uint64(100 * (n + 1)), ShieldsBroken: uint64(n * 7), Mission: mission,
		})
	}
	s.True(proto.Equal(&missionsv1.ListFlagDaysResponse{Days: []*missionsv1.FlagDays{{Day: s.protoOf(day), Flags: want}}}, answer),
		"%v", answer)
}

func (s *testSuite) TestAFlagDayWithNoMissionAndADayNobodyPlayedAreRead() {
	quiet, played := s.told(0), s.told(1)
	s.played(played, "fr", flagmissions.DaySnapshot{Takes: 12})

	answer, err := s.query().FlagDays(s.T().Context(), firstStart, now)

	s.Require().NoError(err)
	s.True(proto.Equal(&missionsv1.ListFlagDaysResponse{Days: []*missionsv1.FlagDays{
		{Day: s.protoOf(quiet)},
		{Day: s.protoOf(played), Flags: []*missionsv1.FlagDay{{CountryId: "fr", TilesTaken: 12}}},
	}}, answer), "%v", answer)
}

func (s *testSuite) TestOnlyTheDaysThatStartedInTheWindowAreReadOldestFirstWithTheirFlagsByCode() {
	before, first, second, after := s.told(0), s.told(1), s.told(2), s.told(3)
	for _, day := range []days.Day{before, second, first, after} {
		for _, flag := range []string{"se", "ie"} {
			s.played(day, flag, flagmissions.DaySnapshot{Takes: 1})
		}
	}

	answer, err := s.query().FlagDays(s.T().Context(), firstStart.Add(time.Hour), firstStart.Add(72*time.Hour))

	s.Require().NoError(err)
	flags := []*missionsv1.FlagDay{{CountryId: "ie", TilesTaken: 1}, {CountryId: "se", TilesTaken: 1}}
	s.True(proto.Equal(&missionsv1.ListFlagDaysResponse{Days: []*missionsv1.FlagDays{
		{Day: s.protoOf(first), Flags: flags},
		{Day: s.protoOf(second), Flags: flags},
	}}, answer), "%v", answer)
}

func (s *testSuite) TestAnUnsetUntilIsNow() {
	s.told(3)

	answer, err := s.query().FlagDays(s.T().Context(), firstStart, time.Time{})

	s.Require().NoError(err)
	s.Empty(answer.GetDays(), "the day starts at now, so not before it")
}

func (s *testSuite) TestAWindowThatIsNoneOrLongerThanAWeekIsRefused() {
	for _, window := range [][2]time.Time{
		{{}, now},
		{now, now},
		{now, firstStart},
		{now.Add(-7*24*time.Hour - time.Nanosecond), now},
	} {
		_, err := s.query().FlagDays(s.T().Context(), window[0], window[1])

		s.ErrorIs(err, flag_days_query.ErrInvalidWindow, window)
	}
}
