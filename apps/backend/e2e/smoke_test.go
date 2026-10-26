package e2e_test

import (
	"context"
	"net/http"
	"regexp"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var generatedName = regexp.MustCompile(`^[A-Z][a-z]+[A-Z][a-z]+[0-9]{2}$`)

// One booted stack for every test: each test plays its own accounts, tiles and flags.
type testSuite struct {
	suite.Suite

	game gameStack
}

func TestSmoke(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupSuite() {
	s.game = startGame(s.T())
}

func (s *testSuite) TestAGuestClicksATileAndEveryScreenSeesIt() {
	t := s.T()
	guest := s.game.newPlayer(t)
	updates := s.game.tileUpdates(t)

	guest.click(1, "fr")

	update := nextTileUpdate(t, updates)
	assert.Equal(t, uint32(1), update.GetTileId())
	assert.Equal(t, "fr", update.GetCountryId())
	holders := planetv1connect.NewClickServiceClient(http.DefaultClient, s.game.baseURL, connect.WithHTTPGet())
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		res, err := holders.GetTileHolder(t.Context(), connect.NewRequest(&planetv1.GetTileHolderRequest{TileId: 1}))
		require.NoError(c, err)
		assert.Equal(c, guest.account(), res.Msg.GetHolder().GetAccountId())
	}, 5*time.Second, 20*time.Millisecond, "the ledger is flushed")
}

func (s *testSuite) TestATakeCountsOnThePlayersStatsAndInTheSeason() {
	t := s.T()
	ada := s.game.newPlayer(t)

	ada.click(2, "pt")

	require.Eventually(t, func() bool { return ada.stats().GetTilesTaken() == 1 }, 5*time.Second, 20*time.Millisecond,
		"planet relays the take to player")
	require.Eventually(t, func() bool { return ada.mySeason().GetTiles() == 1 }, 5*time.Second, 20*time.Millisecond,
		"and to seasons")
	assert.Eventually(t, func() bool { return s.game.roundStandingOf(t, "pt") != nil }, 10*time.Second, 100*time.Millisecond,
		"seasons reads the map from planet at each snapshot")
}

func (s *testSuite) TestAPlayerDoesAMissionAndIsPaid() {
	t := s.T()
	ada := s.game.newPlayer(t)
	ada.link("google-missions")
	ada.pledge("it")
	first := ada.missions()[0]
	bank := first.GetTakeFor()
	require.NotNil(t, bank, "missions asks planet for the pledged flag and the map to draw the day")
	require.Equal(t, "it", bank.GetCountryId())

	for tile := range bank.GetGoal() {
		ada.click(1000+tile, "it")
	}
	require.Eventually(t, func() bool { return ada.missions()[0].GetPaid() },
		10*time.Second, 20*time.Millisecond, "planet relays each take to missions, which pays the mission done")

	reward := first.GetReward().GetCharges()
	require.NotEqual(t, planetv1.BonusKind_BONUS_KIND_UNSPECIFIED, reward.GetKind(), "the reward is drawn with the day")
	assert.Eventually(t, func() bool { return held(t, ada.charges(), reward.GetKind()) == reward.GetAmount() },
		5*time.Second, 20*time.Millisecond, "missions commands planet to grant it")
}

func held(t *testing.T, charges *planetv1.ChargesHeld, kind planetv1.BonusKind) uint32 {
	t.Helper()

	switch kind {
	case planetv1.BonusKind_BONUS_KIND_SHIELDS:
		return charges.GetShields()
	case planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS:
		return charges.GetSpreadClicksLeft()
	case planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS:
		return charges.GetEnclosures()
	case planetv1.BonusKind_BONUS_KIND_BOMB:
		if charges.GetBomb() {
			return 1
		}
		return 0
	case planetv1.BonusKind_BONUS_KIND_REFILL, planetv1.BonusKind_BONUS_KIND_UNSPECIFIED:
		require.Failf(t, "not a charge a mission pays", "%s", kind)
	}
	return 0
}

func (s *testSuite) TestAGuestSignsInWithGoogleAndChoosesAName() {
	t := s.T()
	ada := s.game.newPlayer(t)
	guest := ada.account()

	ada.link("google-ada")

	assert.Equal(t, guest, ada.account(), "the guest keeps its account")
	me := ada.me()
	assert.Equal(t, authv1.AccountKind_ACCOUNT_KIND_LINKED, me.GetKind())
	assert.Equal(t, []authv1.Provider{authv1.Provider_PROVIDER_GOOGLE}, me.GetProviders())
	require.Eventually(t, func() bool { return generatedName.MatchString(ada.profile().GetName()) },
		5*time.Second, 20*time.Millisecond, "player names the account auth signed in")
	require.NoError(t, ada.setName("Ada_L"), "player asks auth whether the account is linked")
	assert.Equal(t, "Ada_L", ada.profile().GetName())
}

func (s *testSuite) TestAPlayerPostsAMessageAndEveryoneReadsItUnderItsName() {
	t := s.T()
	bob := s.game.namedPlayer(t, "google-bob", "Bob_B")

	sent := bob.post("hello")

	assert.Equal(t, "Bob_B", sent.GetAuthorName())
	var read *chatv1.ChatMessage
	for _, message := range s.game.newPlayer(t).history().GetMessages() {
		if message.GetId() == sent.GetId() {
			read = message
		}
	}
	require.NotNil(t, read, "the message is in the history")
	assert.Equal(t, "hello", read.GetText())
	assert.Equal(t, "Bob_B", read.GetAuthorName())
	require.Eventually(t, func() bool { return bob.track("chatter").GetProgress() == 1 }, 5*time.Second, 20*time.Millisecond,
		"chat tells player of each message")
}

func (s *testSuite) TestAGuestReportsABugWithAScreenshotAndTheOperatorOpensIt() {
	t := s.T()
	guest := s.game.newPlayer(t)
	screenshot := file{contentType: "image/png", body: []byte("a screenshot")}
	for range 3 {
		anonymous := connect.NewRequest(&supportv1.ReportBugRequest{Text: "no session"})
		anonymous.Header().Set("X-Real-IP", callerIP)
		_, err := guest.support().ReportBug(t.Context(), anonymous)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "the session check runs before the address's allowance")
	}

	guest.reportBug("the globe is black", guest.uploaded(screenshot))

	note := s.game.noteOf(t, "the globe is black")
	require.NotNil(t, note)
	assert.Equal(t, guest.account(), note.GetAccountId())
	require.Len(t, note.GetAttachments(), 1)
	assert.Equal(t, screenshot, opened(t, note.GetAttachments()[0]))
}

func (s *testSuite) TestADeletedAccountIsForgottenByEveryModule() {
	t := s.T()
	leaving := s.game.namedPlayer(t, "google-leaving", "Leaving")
	leaving.click(3, "es")
	leaving.markSeen(time.Now().Add(-time.Minute))
	leaving.reportBug("a bug before leaving")
	require.Eventually(t, func() bool {
		return leaving.stats().GetTilesTaken() == 1 && leaving.mySeason().GetTiles() == 1 &&
			leaving.missions()[0].GetTakeFor().GetTaken() == 1
	}, 5*time.Second, 20*time.Millisecond)
	require.NotZero(t, leaving.history().GetSeenUntilUnixMs())
	require.Equal(t, leaving.account(), s.game.noteOf(t, "a bug before leaving").GetAccountId())

	leaving.deleteAccount()

	holders := planetv1connect.NewClickServiceClient(http.DefaultClient, s.game.baseURL, connect.WithHTTPGet())
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Empty(c, leaving.profile().GetName(), "player")
		assert.Zero(c, leaving.stats().GetTilesTaken(), "player")
		assert.Zero(c, leaving.mySeason().GetTiles(), "seasons")
		assert.Zero(c, leaving.history().GetSeenUntilUnixMs(), "chat")
		assert.Zero(c, leaving.missions()[0].GetTakeFor().GetTaken(), "missions")
		note := s.game.noteOf(t, "a bug before leaving")
		require.NotNil(c, note, "support keeps the note")
		assert.Empty(c, note.GetAccountId(), "and names no account")
		holder, err := holders.GetTileHolder(t.Context(), connect.NewRequest(&planetv1.GetTileHolderRequest{TileId: 3}))
		require.NoError(c, err)
		assert.Equal(c, "es", holder.Msg.GetHolder().GetCountryId(), "planet keeps the take")
		assert.Empty(c, holder.Msg.GetHolder().GetAccountId(), "and names no account")
	}, 5*time.Second, 20*time.Millisecond, "auth's outbox tells every module")
}

func (s *testSuite) TestAnOperatorSignsInOnTheAdminPageAndGrantsItselfABomb() {
	t := s.T()
	operator := s.game.newPlayer(t)
	operator.link("the-operator")
	operator.becomeOperator()

	_, err := operator.grantFromThePage(operator.account())
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "the game session never opens the admin page")

	operator.adminSignIn("the-operator")
	granted, err := operator.grantFromThePage(operator.account())

	require.NoError(t, err)
	assert.True(t, granted.GetAfter().GetBomb())

	counters, race, err := operator.battleFromThePage()
	names := cpcolls.NewSet[string]()
	for _, metric := range counters.GetMetrics() {
		names.Add(metric.GetName())
	}
	assert.True(t, names.Contains("throttled_clicks"), "cpbootstrap answers the registry every module counts in")
	assert.NotNil(t, race.GetRace(), "seasons answers its race on the admin listener")
	require.NoError(t, err, "the antibot answers the bans its jury passed")
}

func (s gameStack) tileUpdates(t *testing.T) <-chan *planetv1.TileUpdate {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	stream, err := planetv1connect.NewClickServiceClient(http.DefaultClient, s.baseURL).
		ListenForEvents(ctx, connect.NewRequest(&planetv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), "the stream opens with a heartbeat")

	updates := make(chan *planetv1.TileUpdate, 16)
	go func() {
		defer close(updates)
		for stream.Receive() {
			if update := stream.Msg().GetTileUpdate(); update != nil {
				updates <- update
			}
		}
	}()
	return updates
}

func nextTileUpdate(t *testing.T, updates <-chan *planetv1.TileUpdate) *planetv1.TileUpdate {
	t.Helper()

	select {
	case update, open := <-updates:
		require.True(t, open, "the stream ended")
		return update
	case <-time.After(5 * time.Second):
		require.FailNow(t, "no tile update on the stream")
		return nil
	}
}

// A stream opened after a snapshot reads the race at once.
func (s gameStack) roundStandingOf(t *testing.T, country string) *seasonsv1.RoundStanding {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	stream, err := seasonsv1connect.NewSeasonServiceClient(http.DefaultClient, s.baseURL).
		ListenForEvents(ctx, connect.NewRequest(&seasonsv1.ListenForEventsRequest{}))
	if err != nil {
		return nil
	}
	defer func() { _ = stream.Close() }()

	for stream.Receive() {
		for _, standing := range stream.Msg().GetRace().GetRound().GetStandings() {
			if standing.GetCountryId() == country {
				return standing
			}
		}
	}
	return nil
}
