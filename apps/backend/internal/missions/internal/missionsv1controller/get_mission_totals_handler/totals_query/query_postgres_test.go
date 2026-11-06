package totals_query_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/postgres_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_mission_totals_handler/totals_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
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
	store    *postgres_mission_store.Store
}

var (
	ada, bob, cleo = missions.AccountID{15: 1}, missions.AccountID{15: 2}, missions.AccountID{15: 3}
	firstStart     = time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)
	now            = firstStart.Add(72 * time.Hour)
	reward         = missions.Reward{Kind: missions.RewardShields, Amount: 15}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "missions", migrations.FS)
	s.calendar = postgres_day_store.New(s.db)
	s.store = postgres_mission_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) query() *totals_query.PostgresQuery {
	return totals_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now))
}

// The day told n days after the first.
func (s *testSuite) told(n int) days.Day {
	starts := firstStart.Add(time.Duration(n) * 24 * time.Hour)
	day, err := days.DayOf(days.DaySnapshot{
		ID: days.DayID(fmt.Sprintf("season-1/day-%d", n+1)), Season: 1, Number: uint32(n + 1), //nolint:gosec // a few days.
		StartsAt: starts, EndsAt: starts.Add(24 * time.Hour),
	})
	s.Require().NoError(err)
	s.Require().NoError(s.calendar.KeepDay(s.T().Context(), day))
	return day
}

// Keeps the brief of the orders, then the progress and the payments, through the real store.
func (s *testSuite) played(day days.Day, account missions.AccountID, orders []missions.Order, progress []uint32, paid ...int) missions.Brief {
	snapshots := make([]missions.MissionSnapshot, 0, len(orders))
	sheet := missions.SheetSnapshot{
		Progress: map[missions.MissionID]uint32{}, Paid: cpcolls.NewSet[missions.MissionID](),
		Counted: missions.Counted{TakesBelow: missions.LedgerMark{Position: 1}},
	}
	for n, order := range orders {
		id := missions.MissionID{0: account[15], 15: byte(n + 1)}
		snapshots = append(snapshots, missions.MissionSnapshot{ID: id, Order: order, Reward: reward})
		if progress[n] > 0 {
			sheet.Progress[id] = progress[n]
		}
	}
	for _, n := range paid {
		sheet.Paid.Add(snapshots[n].ID)
	}
	brief, err := missions.BriefOf(snapshots)
	s.Require().NoError(err)
	_, err = s.store.KeepBrief(s.T().Context(), day, account, brief)
	s.Require().NoError(err)
	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), day, account, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(sheet)
	}))
	return brief
}

func (s *testSuite) protoOf(day days.Day) *missionsv1.Day {
	snapshot := day.Snapshot()
	return &missionsv1.Day{
		Id: string(snapshot.ID), StartsAt: timestamppb.New(snapshot.StartsAt), EndsAt: timestamppb.New(snapshot.EndsAt),
	}
}

func (s *testSuite) TestEveryKindIsCountedDoneAndPaidAsTheSheetSaysItIs() {
	day := s.told(0)
	drawn := []struct {
		order    missions.Order
		kind     missionsv1.MissionKind
		goal     uint64
		progress uint32
	}{
		{missions.TakeFor{Flag: "fr", Goal: 120}, missionsv1.MissionKind_MISSION_KIND_TAKE_FOR, 120, 120},
		{missions.TakeFrom{Flag: "ie", Goal: 80}, missionsv1.MissionKind_MISSION_KIND_TAKE_FROM, 80, 10},
		{missions.TakeIn{Ground: "ru", Goal: 300}, missionsv1.MissionKind_MISSION_KIND_TAKE_IN, 300, 0},
		{missions.TakeFromIn{Flag: "ie", Ground: "ca", Goal: 40}, missionsv1.MissionKind_MISSION_KIND_TAKE_FROM_IN, 40, 40},
		{missions.CatchBoxes{Goal: 3}, missionsv1.MissionKind_MISSION_KIND_CATCH_BOXES, 3, 3},
		{missions.AnswerQuizzes{Goal: 2}, missionsv1.MissionKind_MISSION_KIND_ANSWER_QUIZZES, 2, 1},
		{missions.BombIn{Ground: "aq", Goal: 1}, missionsv1.MissionKind_MISSION_KIND_BOMB_IN, 1, 1},
		{missions.ShieldTiles{Shields: 10, Goal: 2}, missionsv1.MissionKind_MISSION_KIND_SHIELD_TILES, 2, 0},
		{missions.FortifyTiles{Goal: 20}, missionsv1.MissionKind_MISSION_KIND_FORTIFY_TILES, 20, 20},
	}
	orders, progress := make([]missions.Order, 0, len(drawn)), make([]uint32, 0, len(drawn))
	for _, mission := range drawn {
		orders, progress = append(orders, mission.order), append(progress, mission.progress)
	}
	brief := s.played(day, ada, orders, progress, 0, 4, 6, 8)
	sheet, err := s.store.Sheet(s.T().Context(), day, ada)
	s.Require().NoError(err)

	answer, err := s.query().Totals(s.T().Context(), firstStart, now)

	s.Require().NoError(err)
	want := &missionsv1.DayTotals{Day: s.protoOf(day), Briefs: 1, FinishedAny: 1}
	for n, status := range sheet.Statuses(brief) {
		totals := &missionsv1.KindTotals{Kind: drawn[n].kind, Drawn: 1, Goals: drawn[n].goal, Progress: uint64(status.Progress)}
		if status.Progress > 0 {
			totals.Started = 1
		}
		if status.Done {
			totals.Done = 1
		}
		if status.Paid {
			totals.Paid = 1
		}
		want.Kinds = append(want.Kinds, totals)
	}
	s.True(proto.Equal(&missionsv1.GetMissionTotalsResponse{Days: []*missionsv1.DayTotals{want}}, answer), "%v", answer)
}

func (s *testSuite) TestTheDaysCountTheBriefsAndWhoFinishedOneOrAllOfThem() {
	quiet, played := s.told(0), s.told(1)
	s.played(played, ada, []missions.Order{missions.CatchBoxes{Goal: 3}, missions.CatchBoxes{Goal: 5}}, []uint32{3, 2}, 0)
	s.played(played, bob, []missions.Order{missions.CatchBoxes{Goal: 3}, missions.AnswerQuizzes{Goal: 2}}, []uint32{3, 2}, 0, 1)
	s.played(played, cleo, []missions.Order{missions.CatchBoxes{Goal: 4}}, []uint32{0})

	answer, err := s.query().Totals(s.T().Context(), firstStart, now)

	s.Require().NoError(err)
	s.True(proto.Equal(&missionsv1.GetMissionTotalsResponse{Days: []*missionsv1.DayTotals{
		{Day: s.protoOf(quiet)},
		{Day: s.protoOf(played), Briefs: 3, FinishedAny: 2, FinishedAll: 1, Kinds: []*missionsv1.KindTotals{
			{Kind: missionsv1.MissionKind_MISSION_KIND_CATCH_BOXES, Drawn: 4, Started: 3, Done: 2, Paid: 2, Goals: 15, Progress: 8},
			{Kind: missionsv1.MissionKind_MISSION_KIND_ANSWER_QUIZZES, Drawn: 1, Started: 1, Done: 1, Paid: 1, Goals: 2, Progress: 2},
		}},
	}}, answer), "%v", answer)
}

func (s *testSuite) TestOnlyTheDaysThatStartedInTheWindowAreReadOldestFirst() {
	before, first, second, after := s.told(0), s.told(1), s.told(2), s.told(3)
	for _, day := range []days.Day{before, second, first, after} {
		s.played(day, ada, []missions.Order{missions.CatchBoxes{Goal: 3}}, []uint32{1})
	}

	answer, err := s.query().Totals(s.T().Context(), firstStart.Add(time.Hour), firstStart.Add(72*time.Hour))

	s.Require().NoError(err)
	kinds := []*missionsv1.KindTotals{{Kind: missionsv1.MissionKind_MISSION_KIND_CATCH_BOXES, Drawn: 1, Started: 1, Goals: 3, Progress: 1}}
	s.True(proto.Equal(&missionsv1.GetMissionTotalsResponse{Days: []*missionsv1.DayTotals{
		{Day: s.protoOf(first), Briefs: 1, Kinds: kinds},
		{Day: s.protoOf(second), Briefs: 1, Kinds: kinds},
	}}, answer), "%v", answer)
}

func (s *testSuite) TestAnUnsetUntilIsNow() {
	s.told(3)

	answer, err := s.query().Totals(s.T().Context(), firstStart, time.Time{})

	s.Require().NoError(err)
	s.Empty(answer.GetDays(), "the day starts at now, so not before it")
}

func (s *testSuite) TestAWindowThatIsNoneOrLongerThanAMonthIsRefused() {
	for _, window := range [][2]time.Time{
		{{}, now},
		{now, now},
		{now, firstStart},
		{now.Add(-31*24*time.Hour - time.Nanosecond), now},
	} {
		_, err := s.query().Totals(s.T().Context(), window[0], window[1])

		s.ErrorIs(err, totals_query.ErrInvalidWindow, window)
	}
}
