package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
)

func (s gameStack) chatDesk() chatv1connect.AdminServiceClient {
	return chatv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL)
}

func (s gameStack) supportDesk() supportv1connect.AdminServiceClient {
	return supportv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL)
}

func (s gameStack) message(t *testing.T, id string) *chatv1.GetMessageResponse {
	t.Helper()

	res, err := s.chatDesk().GetMessage(t.Context(), connect.NewRequest(&chatv1.GetMessageRequest{MessageId: id}))
	require.NoError(t, err)
	return res.Msg
}

func (s gameStack) messages(t *testing.T, req *chatv1.ListMessagesRequest) []*chatv1.KeptMessage {
	t.Helper()

	res, err := s.chatDesk().ListMessages(t.Context(), connect.NewRequest(req))
	require.NoError(t, err)
	return res.Msg.GetMessages()
}

func (s gameStack) openReports(t *testing.T) []*supportv1.Report {
	t.Helper()

	res, err := s.supportDesk().ListReports(t.Context(), connect.NewRequest(&supportv1.ListReportsRequest{OpenOnly: true}))
	require.NoError(t, err)
	return res.Msg.GetReports()
}

func TestAnOperatorRemovesAReportedMessageAndClosesTheReport(t *testing.T) {
	game := startGame(t)
	offender, reporter := game.newPlayer(t), game.newPlayer(t)
	insult, err := offender.post()
	require.NoError(t, err)
	_, err = reporter.reply(insult.GetId())
	require.NoError(t, err)
	require.NoError(t, reporter.reportPlayer(insult.GetAuthorAccountId(), supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE,
		"", insult.GetId()))

	open := game.openReports(t)
	require.Len(t, open, 1)
	read := game.message(t, open[0].GetMessageId())
	assert.Equal(t, offender.account(), read.GetAccountId())
	assert.Equal(t, "hello", read.GetText())
	assert.Nil(t, read.GetRemovedAt())

	removed, err := game.chatDesk().RemoveMessage(t.Context(),
		connect.NewRequest(&chatv1.RemoveMessageRequest{MessageId: insult.GetId()}))
	require.NoError(t, err)

	history := reporter.history()
	require.Len(t, history, 2, "the removed message keeps its place")
	assert.True(t, history[0].GetRemoved())
	assert.Empty(t, history[0].GetText(), "with no text")
	assert.True(t, history[1].GetReplyTo().GetRemoved())
	assert.Empty(t, history[1].GetReplyTo().GetText(), "and none in a quote")
	_, err = reporter.react(insult.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "nobody reacts to it")
	read = game.message(t, insult.GetId())
	assert.Equal(t, "hello", read.GetText(), "the operator still reads it")
	assert.WithinDuration(t, removed.Msg.GetRemovedAt().AsTime(), read.GetRemovedAt().AsTime(), time.Microsecond,
		"postgres keeps microseconds")

	_, err = game.supportDesk().DecideReport(t.Context(), connect.NewRequest(&supportv1.DecideReportRequest{
		ReportId: open[0].GetId(), Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_ACTED, Comment: "removed the message",
	}))
	require.NoError(t, err)

	assert.Empty(t, game.openReports(t))
	reports := game.reports(t)
	require.Len(t, reports, 1)
	assert.Equal(t, supportv1.ReportOutcome_REPORT_OUTCOME_ACTED, reports[0].GetDecision().GetOutcome())
	assert.Equal(t, "removed the message", reports[0].GetDecision().GetComment())
}

func TestAnOperatorReadsWhatWasSaidAroundAReportedMessageAPageEachSide(t *testing.T) {
	game := startGame(t)
	offender, reporter := game.newPlayer(t), game.newPlayer(t)
	bait, err := reporter.post()
	require.NoError(t, err)
	insult, err := offender.reply(bait.GetId())
	require.NoError(t, err)
	answer, err := reporter.reply(insult.GetId())
	require.NoError(t, err)

	before := game.messages(t, &chatv1.ListMessagesRequest{BeforeMessageId: insult.GetId()})
	require.Len(t, before, 1)
	assert.Equal(t, bait.GetId(), before[0].GetMessageId())
	assert.Equal(t, reporter.account(), before[0].GetAccountId())
	assert.Equal(t, bait.GetAuthorName(), before[0].GetAuthorName(), "named as the chat names it")

	after := game.messages(t, &chatv1.ListMessagesRequest{AfterMessageId: insult.GetId()})
	require.Len(t, after, 1)
	assert.Equal(t, answer.GetId(), after[0].GetMessageId())
	assert.Equal(t, insult.GetId(), after[0].GetReplyTo().GetMessageId())
	assert.Equal(t, insult.GetAuthorName(), after[0].GetReplyTo().GetAuthorName())

	newest := game.messages(t, &chatv1.ListMessagesRequest{Limit: 2})
	require.Len(t, newest, 2)
	assert.Equal(t, insult.GetId(), newest[0].GetMessageId())
	assert.Equal(t, answer.GetId(), newest[1].GetMessageId())
}

func TestAnOperatorReadsTheMessagesOfAPlayerReportedFromItsCard(t *testing.T) {
	game := startGame(t)
	offender, reporter := game.newPlayer(t), game.newPlayer(t)
	first, err := offender.post()
	require.NoError(t, err)
	between, err := reporter.post()
	require.NoError(t, err)
	second, err := offender.post()
	require.NoError(t, err)
	require.NoError(t, reporter.reportPlayer(offender.account(), supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE, "", ""))

	report := game.openReports(t)[0]
	require.Empty(t, report.GetMessageId(), "a report from the card names no message")

	theirs := game.messages(t, &chatv1.ListMessagesRequest{AccountId: report.GetReportedAccountId()})
	require.Len(t, theirs, 2)
	assert.Equal(t, first.GetId(), theirs[0].GetMessageId())
	assert.Equal(t, second.GetId(), theirs[1].GetMessageId())

	around := game.messages(t, &chatv1.ListMessagesRequest{AfterMessageId: first.GetId(), Limit: 1})
	require.Len(t, around, 1)
	assert.Equal(t, between.GetId(), around[0].GetMessageId(), "the context is everyone's")

	_, err = game.chatDesk().ListMessages(t.Context(),
		connect.NewRequest(&chatv1.ListMessagesRequest{AccountId: "not-an-account"}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestNothingToRemoveReadOrDecideIsNotFound(t *testing.T) {
	game := startGame(t)

	_, err := game.chatDesk().RemoveMessage(t.Context(),
		connect.NewRequest(&chatv1.RemoveMessageRequest{MessageId: "no-such-message"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = game.chatDesk().GetMessage(t.Context(),
		connect.NewRequest(&chatv1.GetMessageRequest{MessageId: "no-such-message"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = game.chatDesk().ListMessages(t.Context(),
		connect.NewRequest(&chatv1.ListMessagesRequest{BeforeMessageId: "no-such-message"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = game.supportDesk().DecideReport(t.Context(), connect.NewRequest(&supportv1.DecideReportRequest{
		ReportId: uuid.NewString(), Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED,
	}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
