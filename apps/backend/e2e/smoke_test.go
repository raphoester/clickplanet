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
	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
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
		assert.Equal(c, guest.on(c).account(), res.Msg.GetHolder().GetAccountId())
	}, 5*time.Second, 20*time.Millisecond, "the ledger is flushed")
}

func (s *testSuite) TestATakeCountsOnThePlayersStatsAndInTheSeason() {
	t := s.T()
	ada := s.game.newPlayer(t)

	ada.click(2, "pt")

	require.EventuallyWithT(t, func(c *assert.CollectT) { assert.EqualValues(c, 1, ada.on(c).stats().GetTilesTaken()) },
		5*time.Second, 20*time.Millisecond, "planet relays the take to player")
	require.EventuallyWithT(t, func(c *assert.CollectT) { assert.EqualValues(c, 1, ada.on(c).myTally().GetTiles()) },
		5*time.Second, 20*time.Millisecond, "and to scoring")
	assert.EventuallyWithT(t, func(c *assert.CollectT) { assert.NotNil(c, s.game.roundStandingOf(tickOf(t, c), "pt")) },
		10*time.Second, 100*time.Millisecond, "scoring reads the map from planet for today's round")
}

func (s *testSuite) TestAGuestDoesAMissionAndIsPaidOnceItSignsIn() {
	t := s.T()
	ada := s.game.newPlayer(t)
	ada.pledge("it")
	first := ada.missions()[0]
	bank := first.GetTakeFor()
	require.NotNil(t, bank, "missions asks planet for the pledged flag and the map to draw the day")
	require.Equal(t, "it", bank.GetCountryId())
	require.NotNil(t, first.GetReward().GetXp(), "a mission pays XP")

	for tile := range bank.GetGoal() {
		ada.click(1000+tile, "it")
	}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.GreaterOrEqual(c, ada.on(c).missions()[0].GetTakeFor().GetTaken(), bank.GetGoal())
	}, 10*time.Second, 20*time.Millisecond, "planet relays each take to missions, which counts a guest's too")
	require.False(t, ada.missions()[0].GetPaid(), "missions asks player whether the account is signed in, in the count's transaction")

	ada.link("google-missions")
	require.EventuallyWithT(t, func(c *assert.CollectT) { assert.True(c, ada.on(c).missions()[0].GetPaid()) },
		10*time.Second, 20*time.Millisecond, "player tells the sign-in (PlayerSignedIn), and missions pays the mission done")

	var goals uint64
	for _, mission := range ada.missions() {
		if mission.GetDone() {
			require.NotNil(t, mission.GetTakeFor(),
				"with no rival of 1,000 tiles the second mission is a take-for too, which the bank's clicks may do; no other kind")
			goals += uint64(mission.GetTakeFor().GetGoal())
		}
	}
	rate := ada.missionRate(missionsv1.MissionKind_MISSION_KIND_TAKE_FOR)
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Equal(c, goals*uint64(rate.GetXp())/uint64(rate.GetPer()), ada.on(c).xp().GetMissionsDone())
	}, 5*time.Second, 20*time.Millisecond, "missions tells the missions done, and progression pays their goals times their kind's rate")
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
	assert.EventuallyWithT(t, func(c *assert.CollectT) { assert.EqualValues(c, 1, ada.on(c).charges().GetRefills()) },
		5*time.Second, 20*time.Millisecond, "player makes the guest a player, and planet grants it a refill")
	require.EventuallyWithT(t, func(c *assert.CollectT) { assert.Regexp(c, generatedName, ada.on(c).profile().GetName()) },
		5*time.Second, 20*time.Millisecond, "player names the account auth signed in")
	require.NoError(t, ada.setName("Ada_L"), "player asks auth whether the account is linked")
	assert.Equal(t, "Ada_L", ada.profile().GetName())
}

func (s *testSuite) TestAPlayerPostsAMessageAndEveryoneReadsItUnderItsName() {
	t := s.T()
	bob := s.game.namedPlayer(t, "google-bob", "Bob_B")

	sent := bob.post("hello")

	assert.Equal(t, bob.account(), sent.GetAuthorAccountId())
	reader := s.game.newPlayer(t)
	var read *chatv1.ChatMessage
	for _, message := range reader.history().GetMessages() {
		if message.GetId() == sent.GetId() {
			read = message
		}
	}
	require.NotNil(t, read, "the message is in the history")
	assert.Equal(t, "hello", read.GetText())
	assert.Equal(t, "Bob_B", reader.nameplate(read.GetAuthorAccountId()).GetName(), "the page names it by its account")
	require.EventuallyWithT(t, func(c *assert.CollectT) { assert.EqualValues(c, 1, bob.on(c).track("chatter").GetProgress()) },
		5*time.Second, 20*time.Millisecond, "chat tells player of each message")
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
	assert.Equal(t, "player@example.com", note.GetReplyAddress())
	require.Len(t, note.GetAttachments(), 1)
	assert.Equal(t, screenshot, opened(t, note.GetAttachments()[0]))
	assert.Contains(t, s.game.openNotes(t), note.GetId(), "a note sent waits for its answer")

	_, err := s.game.supportAdmin().AnswerNote(t.Context(),
		connect.NewRequest(&supportv1.AnswerNoteRequest{NoteId: note.GetId(), Text: "Fixed: reload the page."}))
	require.NoError(t, err)

	answered := s.game.noteOf(t, "the globe is black")
	require.Len(t, answered.GetAnswers(), 1)
	assert.Equal(t, "Fixed: reload the page.", answered.GetAnswers()[0].GetText())
	assert.NotContains(t, s.game.openNotes(t), note.GetId(), "an answered note waits for nothing")

	address := "support+" + note.GetId() + "@clickplanet.lol"
	reply := "From: Ada <player@example.com>\r\nTo: " + address + "\r\nMessage-ID: <reply-1@mail.example.com>\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\nIt works now, thanks!\r\n\r\n" +
		"On Mon, ClickPlanet <" + address + "> wrote:\r\n> Fixed: reload the page.\r\n"
	for range 2 {
		assert.Equal(t, http.StatusAccepted, s.game.mail(t, address, reply), "kept once, then done with")
	}

	replied := s.game.noteOf(t, "the globe is black")
	require.Len(t, replied.GetReplies(), 1)
	assert.Equal(t, "It works now, thanks!", replied.GetReplies()[0].GetText())
	assert.Equal(t, "player@example.com", replied.GetReplies()[0].GetFrom())
	assert.Contains(t, s.game.openNotes(t), note.GetId(), "the player wrote last")
}

func (s *testSuite) TestADeletedAccountIsForgottenByEveryModule() {
	t := s.T()
	leaving := s.game.namedPlayer(t, "google-leaving", "Leaving")
	leaving.click(3, "es")
	leaving.markSeen(time.Now().Add(-time.Minute))
	leaving.reportBug("a bug before leaving")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.EqualValues(c, 1, leaving.on(c).stats().GetTilesTaken(), "progression")
		assert.EqualValues(c, 1, leaving.on(c).myTally().GetTiles(), "scoring")
		assert.EqualValues(c, 1, leaving.on(c).missions()[0].GetTakeFor().GetTaken(), "missions")
	}, 5*time.Second, 20*time.Millisecond)
	require.NotZero(t, leaving.history().GetSeenUntilUnixMs())
	require.Equal(t, leaving.account(), s.game.noteOf(t, "a bug before leaving").GetAccountId())

	account := leaving.account()
	// No quiz comes within a test: the answer it gave is written as the quizzes write one.
	quizzes := s.game.schema(t, "quizzes")
	_, err := quizzes.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO quiz_answers (question_id, account_id, outcome, answer_ms, answered_at)
		VALUES ('capital:ee', $1, 'right', 2500, now())`, account)
	require.NoError(t, err)
	answerer := func(t check) *string {
		var kept *string
		require.NoError(t, quizzes.On(t.Context()).QueryRowContext(t.Context(),
			`SELECT account_id::text FROM quiz_answers WHERE question_id = 'capital:ee' AND answer_ms = 2500`).Scan(&kept))
		return kept
	}
	require.Equal(t, &account, answerer(t))

	watcher := s.game.newPlayer(t)
	card, err := watcher.playerCard(account)
	require.NoError(t, err)
	require.Equal(t, "Leaving", card.GetName())

	leaving.deleteAccount()

	assert.Equal(t, uint64(1), leaving.stats().GetTilesTaken(), "kept whole for the grace period")
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := watcher.playerCard(account)
		assert.Equal(c, connect.CodeNotFound, connect.CodeOf(err), "the other players see it deleted at once")
	}, deletionGrace, 20*time.Millisecond, "auth's outbox tells player of the request")

	holders := planetv1connect.NewClickServiceClient(http.DefaultClient, s.game.baseURL, connect.WithHTTPGet())
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Empty(c, leaving.on(c).profile().GetName(), "player")
		assert.Zero(c, leaving.on(c).stats().GetTilesTaken(), "progression")
		assert.Zero(c, leaving.on(c).myTally().GetTiles(), "scoring")
		assert.Zero(c, leaving.on(c).history().GetSeenUntilUnixMs(), "chat")
		assert.Zero(c, leaving.on(c).missions()[0].GetTakeFor().GetTaken(), "missions")
		note := s.game.noteOf(tickOf(t, c), "a bug before leaving")
		require.NotNil(c, note, "support keeps the note")
		assert.Empty(c, note.GetAccountId(), "and names no account")
		assert.Empty(c, note.GetReplyAddress(), "nor where to answer")
		holder, err := holders.GetTileHolder(t.Context(), connect.NewRequest(&planetv1.GetTileHolderRequest{TileId: 3}))
		require.NoError(c, err)
		assert.Equal(c, "es", holder.Msg.GetHolder().GetCountryId(), "planet keeps the take")
		assert.Empty(c, holder.Msg.GetHolder().GetAccountId(), "and names no account")
		assert.Nil(c, answerer(tickOf(t, c)), "the quizzes keep the answer and name no account")
	}, deletionGrace+5*time.Second, 20*time.Millisecond, "the purge deletes it once the grace period ends, auth's outbox tells player, and player's every module")
}

func (s *testSuite) TestAnAccountSignedInToWithinTheGracePeriodComesBack() {
	t := s.T()
	leaving := s.game.namedPlayer(t, "google-coming-back", "Back")
	account := leaving.account()
	watcher := s.game.newPlayer(t)

	leaving.deleteAccount()
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := watcher.playerCard(account)
		assert.Equal(c, connect.CodeNotFound, connect.CodeOf(err))
	}, deletionGrace, 20*time.Millisecond, "hidden first")

	back := s.game.newPlayer(t).signIn("google-coming-back")

	assert.True(t, back.GetRestored())
	assert.Equal(t, account, back.GetAccountId())
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		card, err := watcher.playerCard(account)
		require.NoError(c, err)
		assert.Equal(c, "Back", card.GetName())
	}, 5*time.Second, 20*time.Millisecond, "the other players see it again")
	assert.Never(t, func() bool {
		_, err := watcher.playerCard(account)
		return err != nil
	}, deletionGrace+time.Second, 50*time.Millisecond, "and the purge leaves it alone")
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
	assert.Equal(t, uint32(1), granted.GetAfter().GetBombs())
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

// A stream opened reads today's round at once.
func (s gameStack) roundStandingOf(t check, country string) *scoringv1.RoundStanding {
	t.Helper()

	today := s.today(t)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	stream, err := scoringv1connect.NewScoringServiceClient(http.DefaultClient, s.baseURL).
		ListenForEvents(ctx, connect.NewRequest(&scoringv1.ListenForEventsRequest{BookId: today.book, RoundIds: []string{today.round}}))
	if err != nil {
		return nil
	}
	defer func() { _ = stream.Close() }()

	for stream.Receive() {
		for _, standing := range stream.Msg().GetRound().GetStandings() {
			if standing.GetCountryId() == country {
				return standing
			}
		}
	}
	return nil
}
