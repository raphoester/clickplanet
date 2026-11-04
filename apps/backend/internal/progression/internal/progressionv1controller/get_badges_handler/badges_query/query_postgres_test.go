package badges_query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures/postgres_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates/postgres_nameplate_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_badges_handler/badges_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/postgres_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing/postgres_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db         *cppg.Postgres
	stats      *postgres_stats_store.Store
	nameplates *postgres_nameplate_store.Store
	titles     *postgres_title_store.Store
	worn       *postgres_worn_title_store.Store
}

var (
	ada   = accounts.AccountID{15: 1}
	bob   = accounts.AccountID{15: 2}
	guest = accounts.AccountID{15: 3}
	today = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "progression", migrations.FS)
	s.stats = postgres_stats_store.New(s.db)
	s.nameplates = postgres_nameplate_store.New(s.db)
	s.titles = postgres_title_store.New(s.db)
	s.worn = postgres_worn_title_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.named(ada)
	s.named(bob)
}

func (s *testSuite) named(account accounts.AccountID) {
	s.Require().NoError(s.nameplates.Keep(s.T().Context(), account))
}

func (s *testSuite) took(account accounts.AccountID, days ...time.Time) {
	for position, day := range days {
		s.Require().NoError(s.stats.Record(s.T().Context(), account, stats.TakeAt(uint64(position), 0, day)))
	}
}

func (s *testSuite) query() *badges_query.PostgresQuery {
	return badges_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), cptime.NewFixedClock(today))
}

func (s *testSuite) answer(accounts ...accounts.AccountID) *progressionv1.GetBadgesResponse {
	answer, err := s.query().Badges(s.T().Context(), accounts)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) badges(accounts ...accounts.AccountID) map[string]*progressionv1.Badge {
	by := make(map[string]*progressionv1.Badge)
	for _, badge := range s.answer(accounts...).GetBadges() {
		by[badge.GetAccountId()] = badge
	}
	return by
}

func (s *testSuite) TestEachBadgeIsTheStreakAsOfTodayAndTheTitleWorn() {
	s.took(ada, today.AddDate(0, 0, -1), today)
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"og", "settler"}, bob: {"og", "settler"}}, today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), ada, "og", today))

	badges := s.badges(ada, bob)

	s.Equal(uint32(2), badges[ada.String()].GetStreak())
	s.Equal("og", badges[ada.String()].GetWornTitle().GetId())
	s.Equal("OG", badges[ada.String()].GetWornTitle().GetName())
	s.Zero(badges[bob.String()].GetStreak())
	s.Equal("settler", badges[bob.String()].GetWornTitle().GetId(), "no choice wears the first rank shown")
	s.Equal("conquest", badges[bob.String()].GetWornTitle().GetRank().GetTrackId())
}

func (s *testSuite) TestAStreakBrokenSinceReadsZero() {
	s.took(ada, today.AddDate(0, 0, -3), today.AddDate(0, 0, -2))
	s.took(bob, today.AddDate(0, 0, -1))

	badges := s.badges(ada, bob)

	s.NotContains(badges, ada.String(), "a broken streak and no title is no badge")
	s.Equal(uint32(1), badges[bob.String()].GetStreak(), "a streak whose last day is yesterday still runs")
}

func (s *testSuite) TestAnAccountWithNothingToShowIsLeftOut() {
	s.Require().NoError(s.stats.Record(s.T().Context(), ada, stats.MessageAt(today)))

	s.Empty(s.badges(ada, bob), "a post is no take: no streak, and nobody holds a title")
}

func (s *testSuite) TestAGuestWearsNoBadgeHoweverLongItsStreak() {
	s.took(guest, today.AddDate(0, 0, -1), today)
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{guest: {"og"}}, today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), guest, "og", today))

	s.Empty(s.badges(guest))
}

func (s *testSuite) TestAnUnknownOrDeletedAccountIsLeftOut() {
	s.took(ada, today)
	s.took(bob, today)
	s.Require().NoError(s.stats.Forget(s.T().Context(), bob))
	s.Require().NoError(s.nameplates.Forget(s.T().Context(), bob))

	s.Equal([]string{ada.String()}, accountsOf(s.answer(ada, bob, accounts.AccountID{15: 9})))
}

func (s *testSuite) decide(account accounts.AccountID, decision departures.Decision) {
	s.Require().NoError(postgres_departure_store.New(s.db).Change(s.T().Context(), account,
		func(d departures.Departure) departures.Departure { return d.Decided(decision) }))
}

func (s *testSuite) TestAnAccountAskedToBeDeletedShowsNoBadgeUntilItComesBack() {
	s.took(ada, today)
	s.took(bob, today)
	s.decide(ada, departures.Leaves(today))

	s.Equal([]string{bob.String()}, accountsOf(s.answer(ada, bob)), "the other players see it gone")

	s.decide(ada, departures.Returns(today.Add(time.Hour)))

	s.Equal([]string{ada.String(), bob.String()}, accountsOf(s.answer(ada, bob)), "back with its badge")
}

func (s *testSuite) TestARepeatedAccountIsAnsweredOnceInTheOrderAsked() {
	s.took(ada, today)
	s.took(bob, today)

	s.Equal([]string{bob.String(), ada.String()}, accountsOf(s.answer(bob, ada, bob, guest, ada)))
}

func (s *testSuite) TestAskingAboutNobodyAnswersNobody() {
	s.Empty(s.answer().GetBadges())
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query().Badges(ctx, []accounts.AccountID{ada})

	s.Error(err)
}

// The rules the commands apply, over the same rows: a username's streak as of today (Stats.AsOf) and the title its
// wardrobe shows as worn (wearing.WornOf), and nothing for a guest.
func (s *testSuite) badgeOf(account accounts.AccountID, wardrobe wearing.Wardrobe) *progressionv1.Badge {
	named, err := s.nameplates.Named(s.T().Context(), account)
	s.Require().NoError(err)
	if !named {
		return nil
	}
	counted, err := s.stats.Stats(s.T().Context(), account)
	if errors.Is(err, stats.ErrNoStats) {
		counted = stats.NewStats(account)
	} else {
		s.Require().NoError(err)
	}
	showcase, err := wardrobe.Showcase(s.T().Context(), account)
	s.Require().NoError(err)

	badge := &progressionv1.Badge{
		AccountId: account.String(),
		Streak:    counted.AsOf(stats.DayOf(today)).Streak().Days(),
		WornTitle: progressionmessage.Title(showcase.Worn()),
	}
	if badge.GetStreak() == 0 && badge.GetWornTitle() == nil {
		return nil
	}
	return badge
}

func (s *testSuite) TestEachBadgeIsWhatTheRulesShowBesideTheName() {
	cy, dan, eve := accounts.AccountID{15: 4}, accounts.AccountID{15: 5}, accounts.AccountID{15: 6}
	s.named(cy)
	s.named(dan)
	s.took(ada, today.AddDate(0, 0, -1), today)
	s.took(bob, today.AddDate(0, 0, -3), today.AddDate(0, 0, -2))
	s.took(guest, today.AddDate(0, 0, -1), today)
	s.took(dan, today.AddDate(0, 0, 1))
	s.Require().NoError(s.stats.Record(s.T().Context(), cy, stats.MessageAt(today)))
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{
		ada: {"og", "settler"}, bob: {"settler", "raider", "loyal"}, guest: {"og"}, dan: {"retired", "og"},
	}, today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), ada, "og", today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), bob, "settler", today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), guest, "og", today))

	everyone := []accounts.AccountID{ada, bob, guest, cy, dan, eve}
	catalog := titles.NewCatalog()
	wardrobe := wearing.NewWardrobe(s.worn, titles.NewBook(s.titles, catalog), catalog)
	badges := s.badges(everyone...)

	for _, account := range everyone {
		want := s.badgeOf(account, wardrobe)
		badge, shown := badges[account.String()]
		if want == nil {
			s.False(shown, "%s shows nothing beside its name, so it has no badge, got %v", account, badge)
			continue
		}
		s.True(proto.Equal(want, badge), "%s: the rules show %v, GetBadges answers %v", account, want, badge)
	}
}

func accountsOf(answer *progressionv1.GetBadgesResponse) []string {
	ids := make([]string, 0, len(answer.GetBadges()))
	for _, badge := range answer.GetBadges() {
		ids = append(ids, badge.GetAccountId())
	}
	return ids
}
