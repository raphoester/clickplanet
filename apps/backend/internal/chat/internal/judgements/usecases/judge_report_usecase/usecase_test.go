package judge_report_usecase_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/inmemory_judgement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/usecases/judge_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/inmemory_moderation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/inmemory_mute_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now      = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)
	raider   = messages.AccountID{15: 1}
	reporter = messages.AccountID{15: 2}
	other    = messages.AccountID{15: 3}
	first    = judgements.ReportID{15: 1}
	second   = judgements.ReportID{15: 2}
)

type testSuite struct {
	suite.Suite

	clock         *cptime.FixedClock
	store         *inmemory_judgement_storage.Storage
	conversations *fakeConversations
	authors       *fakeAuthors
	judge         *fakeJudge
	history       *inmemory_moderation_storage.Storage
	mutes         *inmemory_mute_storage.Storage
	removed       *recordedRemovals
	outbox        *cpatomic.FakeOutbox
	cap           int
	useCase       *judge_report_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	s.clock = cptime.NewFixedClock(now)
	s.store = inmemory_judgement_storage.New()
	s.conversations = &fakeConversations{lines: []judgements.Line{
		{ID: "m-1", Account: other, Room: messages.World, SentAt: now.Add(-2 * time.Minute), Text: "gg France"},
		{ID: "m-2", Account: raider, Room: messages.World, SentAt: now.Add(-time.Minute), Text: "<a slur>"},
		{ID: "m-3", Account: reporter, Room: messages.World, SentAt: now, Text: "reported"},
	}}
	s.authors = &fakeAuthors{}
	s.judge = &fakeJudge{answer: judgements.Answer{
		Category: judgements.CategoryOffence, Offending: []messages.MessageID{"m-2"}, What: "a slur",
		Spend: judgements.Spend{Model: "claude-haiku-5-5", Input: 3000, Output: 300},
	}}
	s.history = inmemory_moderation_storage.New()
	s.mutes = inmemory_mute_storage.New()
	s.removed = &recordedRemovals{}
	s.outbox = &cpatomic.FakeOutbox{}
	s.cap = 10
	s.build()
}

func (s *testSuite) build() {
	window := messages.NewWindow(200, 30*24*time.Hour)
	tx := cpatomic.FakeTransactioner{}
	s.useCase = judge_report_usecase.New(judge_report_usecase.Ports{
		Judgements:    s.store,
		Conversations: s.conversations,
		Authors:       s.authors,
		History:       s.history,
		Judge:         s.judge,
		Calls: cpratelimit.New("judge", cpratelimit.Config{
			PerSecond: float64(s.cap) / time.Hour.Seconds(), Burst: s.cap,
		}, s.clock),
		Transactioner: tx,
		Remover: remove_message_usecase.New(tx, s.conversations, s.removed, s.history, &recordedFeed{}, window,
			s.clock),
		Muter:  mute_usecase.New(tx, s.mutes, s.history, s.conversations, &mutes.SequentialIDs{}, s.clock),
		Outbox: s.outbox,
	}, window, s.clock)
}

func reportOf(id judgements.ReportID, message messages.MessageID) judgements.Report {
	return judgements.Report{
		ID: id, Reporter: reporter, Reported: raider, Reason: judgements.ReasonChatAbuse, Message: message, At: now,
	}
}

func (s *testSuite) judged(report judgements.Report) judge_report_usecase.Out {
	out, err := s.useCase.Execute(s.T().Context(), report)
	s.Require().NoError(err)
	return out
}

func (s *testSuite) decisions() []*supportv1.DecideReport {
	var decided []*supportv1.DecideReport
	for _, recorded := range s.outbox.Recorded() {
		if decision, ok := recorded.(*supportv1.DecideReport); ok {
			decided = append(decided, decision)
		}
	}
	return decided
}

func (s *testSuite) offence(at time.Time, reason string) {
	message := messages.Restored(s.T(), messages.Snapshot{ID: "earlier", SentAt: at, Account: raider, Text: "earlier"})
	s.Require().NoError(s.history.Record(s.T().Context(),
		moderation.NewRemoval(message, at, moderation.Reason(reason), judgements.By)))
}

func (s *testSuite) TestAnOffenceRemovesMutesAndClosesTheReportInOneGo() {
	s.offence(now.Add(-time.Hour), "offence 1 (chat): an insult")

	out := s.judged(reportOf(first, "m-2"))

	s.Equal(judgements.Acted, out.Judgement.TakeSnapshot().Verdict)
	s.Equal(int64(3000), out.Spend.Input)
	s.Equal([]messages.MessageID{"m-2"}, s.removed.ids)

	kept := s.mutes.Kept()
	s.Require().Len(kept, 1)
	s.Equal(raider, kept[0].Caller().Account())
	s.Equal(mutes.Chat, kept[0].Kind())
	s.Equal(15*time.Minute, kept[0].Duration())

	actions, err := s.history.ActionsOn(s.T().Context(), raider, now)
	s.Require().NoError(err)
	s.Require().Len(actions, 2)
	for _, action := range actions {
		s.Equal(moderation.Reason("offence 2 (chat): a slur"), action.Reason)
		s.Equal("Claude judge", action.By.Client(), "the history tells the judge apart")
		s.Equal(messages.NoAccount, action.By.Account())
	}

	s.Require().Len(s.outbox.Recorded(), 2)
	s.True(proto.Equal(&chatv1.AccountMuted{
		MuteId: kept[0].ID().String(), AccountId: raider.String(), MutedAt: timestamppb.New(now),
		Duration: durationpb.New(15 * time.Minute), Kind: chatv1.MuteKind_MUTE_KIND_CHAT,
	}, s.outbox.Recorded()[0]), "announced once the transaction commits")
	s.True(proto.Equal(&supportv1.DecideReport{
		ReportId: first.String(), Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_ACTED,
		Comment: "Claude judge: offence 2 (chat): a slur; 1 message removed, muted 15 minutes",
	}, s.outbox.Recorded()[1]), "%v", s.outbox.Recorded()[1])
}

func (s *testSuite) TestNoOffenceDismissesTheReportAndTouchesNothing() {
	s.judge.answer = judgements.Answer{Category: judgements.CategoryNone, Why: "banter among rivals"}

	out := s.judged(reportOf(first, "m-2"))

	s.Equal(judgements.Dismissed, out.Judgement.TakeSnapshot().Verdict)
	s.Empty(s.removed.ids)
	s.Empty(s.mutes.Kept())
	s.Require().Len(s.decisions(), 1)
	s.Equal(supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED, s.decisions()[0].GetOutcome())
	s.Equal("Claude judge: no offence: banter among rivals", s.decisions()[0].GetComment())
}

func (s *testSuite) TestAReportToldAgainIsJudgedOnce() {
	s.judged(reportOf(first, "m-2"))

	again := s.judged(reportOf(first, "m-2"))

	s.True(again.Again)
	s.Equal(1, s.judge.calls)
	s.Len(s.decisions(), 1)
	s.Len(s.removed.ids, 1)
}

func (s *testSuite) TestManyReportsOfOneMessageAreOneJudgement() {
	s.judged(reportOf(first, "m-2"))

	out := s.judged(reportOf(second, "m-2"))

	s.Equal(1, s.judge.calls, "the model is asked once per message")
	s.Len(s.removed.ids, 1)
	s.Empty(s.mutes.Kept(), "a first offence, counted once")
	s.Equal(judgements.Acted, out.Judgement.TakeSnapshot().Verdict)
	s.Require().Len(s.decisions(), 2)
	s.Equal(second.String(), s.decisions()[1].GetReportId())
	s.Equal("Claude judge: as report "+first.String()+": offence 1 (chat): a slur; 1 message removed",
		s.decisions()[1].GetComment())
}

func (s *testSuite) TestWhatIsNeverActedOnIsLeftOpenAndFlagged() {
	for _, category := range []judgements.Category{
		judgements.CategoryChildSafety, judgements.CategorySelfHarm, judgements.CategoryCheating, judgements.CategoryUnsure,
	} {
		s.SetupTest()
		s.judge.answer = judgements.Answer{Category: category, Offending: []messages.MessageID{"m-2"}, Why: "an adult asks"}

		out := s.judged(reportOf(first, "m-2"))

		s.Equal(judgements.LeftOpen, out.Judgement.TakeSnapshot().Verdict, category)
		s.True(out.Judgement.TakeSnapshot().Flagged, category)
		s.Empty(s.removed.ids, category)
		s.Empty(s.mutes.Kept(), category)
		s.Empty(s.outbox.Recorded(), "the report stays open: %s", category)
	}
}

func (s *testSuite) TestACheatingReportIsLeftOpenUnread() {
	report := reportOf(first, judgements.NoMessage)
	report.Reason = judgements.ReasonCheating

	out := s.judged(report)

	s.Equal(judgements.LeftOpen, out.Judgement.TakeSnapshot().Verdict)
	s.True(out.Judgement.TakeSnapshot().Flagged)
	s.Zero(s.conversations.reads)
	s.Zero(s.judge.calls)
}

func (s *testSuite) TestAMessagePastTheRetentionIsDismissedAndTheModelNotAsked() {
	out := s.judged(reportOf(first, "m-pruned"))

	s.Equal(judgements.Dismissed, out.Judgement.TakeSnapshot().Verdict)
	s.Zero(s.judge.calls)
	s.Require().Len(s.decisions(), 1)
	s.Contains(s.decisions()[0].GetComment(), "past the chat's retention")
}

func (s *testSuite) TestAMessageRemovedAlreadyClosesTheReportAsActed() {
	s.conversations.lines[1].Removed = true

	out := s.judged(reportOf(first, "m-2"))

	s.Equal(judgements.Acted, out.Judgement.TakeSnapshot().Verdict)
	s.Zero(s.judge.calls)
	s.Empty(s.removed.ids)
}

func (s *testSuite) TestAMessageOfAnotherPlayerIsLeftOpen() {
	out := s.judged(reportOf(first, "m-1"))

	s.Equal(judgements.LeftOpen, out.Judgement.TakeSnapshot().Verdict)
	s.Zero(s.judge.calls)
}

func (s *testSuite) TestTheOperatorsOwnAccountIsNeverActedOn() {
	s.authors.admin = true

	out := s.judged(reportOf(first, "m-2"))

	s.Equal(judgements.LeftOpen, out.Judgement.TakeSnapshot().Verdict)
	s.Zero(s.judge.calls)
	s.Empty(s.removed.ids)
}

func (s *testSuite) TestOnceTheHourlyCapIsSpentTheRestIsLeftOpen() {
	s.cap = 1
	s.build()
	s.conversations.lines = append(s.conversations.lines,
		judgements.Line{ID: "m-4", Account: raider, Room: messages.World, SentAt: now, Text: "<another slur>"})
	s.judged(reportOf(first, "m-2"))

	out := s.judged(reportOf(second, "m-4"))

	s.Equal(1, s.judge.calls)
	s.Equal(judgements.LeftOpen, out.Judgement.TakeSnapshot().Verdict)
	s.True(out.Judgement.TakeSnapshot().Flagged)
	s.Contains(out.Judgement.TakeSnapshot().Reason, "cap")

	s.clock.Advance(time.Hour)
	third := s.judged(reportOf(judgements.ReportID{15: 3}, "m-4"))
	s.Equal(2, s.judge.calls, "a message left before it was judged is judged at the next report")
	s.Equal(judgements.Acted, third.Judgement.TakeSnapshot().Verdict)
}

func (s *testSuite) TestWithNoKeyEveryReportIsLeftOpen() {
	s.judge.err = judgements.ErrNoJudge

	out := s.judged(reportOf(first, "m-2"))

	s.Equal(judgements.LeftOpen, out.Judgement.TakeSnapshot().Verdict)
	s.False(out.Judgement.TakeSnapshot().Judged)
	s.Empty(s.outbox.Recorded())
}

func (s *testSuite) TestARefusalIsLeftOpenAndStandsForTheMessage() {
	s.judge.err = judgements.ErrUnjudgeable

	s.judged(reportOf(first, "m-2"))
	out := s.judged(reportOf(second, "m-2"))

	s.Equal(1, s.judge.calls)
	s.Equal(judgements.LeftOpen, out.Judgement.TakeSnapshot().Verdict)
}

func (s *testSuite) TestAFailureOfTheModelIsTriedAgainAndKeepsNothing() {
	s.judge.err = errors.New("overloaded")

	_, err := s.useCase.Execute(s.T().Context(), reportOf(first, "m-2"))
	s.Require().Error(err)
	s.Empty(s.store.Kept())
	s.Empty(s.outbox.Recorded())

	s.judge.err = nil
	out := s.judged(reportOf(first, "m-2"))
	s.Equal(judgements.Acted, out.Judgement.TakeSnapshot().Verdict)
}

func (s *testSuite) TestAFailureToKeepTheJudgementClosesNothing() {
	cause := errors.New("postgres is down")
	s.store.FailWith(cause)

	_, err := s.useCase.Execute(s.T().Context(), reportOf(first, "m-2"))

	s.Require().ErrorIs(err, cause)
	s.Empty(s.outbox.Recorded())
}

type fakeConversations struct {
	lines []judgements.Line
	reads int
}

func (f *fakeConversations) Conversation(
	_ context.Context, message messages.MessageID, account messages.AccountID, _ time.Time,
) (judgements.Conversation, error) {
	f.reads++
	var conversation judgements.Conversation
	for i, line := range f.lines {
		if line.ID == message {
			reported := line
			conversation.Reported = &reported
			conversation.Before, conversation.After = slices.Clone(f.lines[:i]), slices.Clone(f.lines[i+1:])
		}
		if line.Account == account {
			conversation.Own = append(conversation.Own, line)
		}
	}
	if message != judgements.NoMessage && conversation.Reported == nil {
		return judgements.Conversation{}, messages.ErrNotKept
	}
	return conversation, nil
}

func (f *fakeConversations) KeptMessage(_ context.Context, id messages.MessageID, _ time.Time) (messages.Message, error) {
	for _, line := range f.lines {
		if line.ID == id {
			return messages.Snapshot{ //nolint:wrapcheck // a fake answers what the store would.
				ID: id, SentAt: line.SentAt, Account: line.Account, Room: line.Room, Text: line.Text, Gif: line.Gif,
			}.Restore()
		}
	}
	return messages.Message{}, messages.ErrNotKept
}

func (f *fakeConversations) LatestAddress(context.Context, messages.AccountID) (string, error) {
	return "203.0.113.7", nil
}

type fakeAuthors struct{ admin bool }

func (f *fakeAuthors) Author(context.Context, messages.AccountID) (messages.Author, error) {
	if f.admin {
		return messages.AdminOf("raphael"), nil
	}
	return messages.GuestOf("guest_0b1c2d"), nil
}

type fakeJudge struct {
	answer judgements.Answer
	err    error
	calls  int
}

func (f *fakeJudge) Judge(context.Context, judgements.Report, judgements.Conversation) (judgements.Answer, error) {
	f.calls++
	if f.err != nil {
		return judgements.Answer{}, f.err
	}
	return f.answer, nil
}

type recordedRemovals struct{ ids []messages.MessageID }

func (r *recordedRemovals) Save(ctx context.Context, removal removals.Snapshot) error {
	cpatomic.OnCommit(ctx, func() { r.ids = append(r.ids, removal.MessageID) })
	return nil
}

type recordedFeed struct{}

func (recordedFeed) Publish(feed.Update) {}

func (s *testSuite) TestAReadThatFailsSpendsNothingOfTheCap() {
	s.cap = 1
	s.build()
	cause := errors.New("postgres is down")
	s.history.FailWith(cause)

	_, err := s.useCase.Execute(s.T().Context(), reportOf(first, "m-2"))
	s.Require().ErrorIs(err, cause)

	s.history.FailWith(nil)
	out := s.judged(reportOf(first, "m-2"))
	s.Equal(judgements.Acted, out.Judgement.TakeSnapshot().Verdict, "the bus tried it again, and the cap still had its call")
}
