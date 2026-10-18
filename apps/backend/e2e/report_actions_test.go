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

func TestNothingToRemoveReadOrDecideIsNotFound(t *testing.T) {
	game := startGame(t)

	_, err := game.chatDesk().RemoveMessage(t.Context(),
		connect.NewRequest(&chatv1.RemoveMessageRequest{MessageId: "no-such-message"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = game.chatDesk().GetMessage(t.Context(),
		connect.NewRequest(&chatv1.GetMessageRequest{MessageId: "no-such-message"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = game.supportDesk().DecideReport(t.Context(), connect.NewRequest(&supportv1.DecideReportRequest{
		ReportId: uuid.NewString(), Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED,
	}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
