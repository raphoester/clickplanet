package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
)

var reportedMessage = regexp.MustCompile(`The message reported: \[([^\]]+)\]`)

// A stand-in for the Messages API: every message reported is an offence, so a report travels the whole way.
func startJudge(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request)) || len(request.Messages) == 0 {
			http.Error(w, "no prompt", http.StatusBadRequest)
			return
		}
		var offending []string
		if found := reportedMessage.FindStringSubmatch(request.Messages[0].Content[0].Text); found != nil {
			offending = append(offending, found[1])
		}
		answer, err := json.Marshal(map[string]any{
			"category": "offence", "severe": false, "offending": offending, "what": "an insult", "why": "It insults a player.",
		})
		assert.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-haiku-5-5",
			"content":     []map[string]any{{"type": "text", "text": string(answer)}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 3000, "output_tokens": 100},
		}))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func (p *gamer) reportPlayer(account string, message string) {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.ReportPlayerRequest{
		AccountId: account, MessageId: message, Reason: supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE,
	})
	p.send(req.Header())
	_, err := p.support().ReportPlayer(p.t.Context(), req)
	require.NoError(p.t, err)
}

func (s *testSuite) TestAReportIsJudgedTheMessageRemovedAndTheReportClosed() {
	t := s.T()
	raider, reporter := s.game.newPlayer(t), s.game.newPlayer(t)
	said := raider.post(fmt.Sprintf("you are all idiots %d", time.Now().UnixNano()))

	reporter.reportPlayer(raider.account(), said.GetId())

	reports := supportv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
	var decided *supportv1.ReportDecision
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res, err := reports.ListReports(t.Context(), connect.NewRequest(&supportv1.ListReportsRequest{}))
		require.NoError(c, err)
		for _, report := range res.Msg.GetReports() {
			if report.GetMessageId() == said.GetId() {
				decided = report.GetDecision()
			}
		}
		require.NotNil(c, decided)
	}, 15*time.Second, 50*time.Millisecond,
		"support tells PlayerReported, the chat judges it, and its outbox sends DecideReport back to support")
	assert.Equal(t, supportv1.ReportOutcome_REPORT_OUTCOME_ACTED, decided.GetOutcome())
	assert.True(t, strings.HasPrefix(decided.GetComment(), "Claude judge: offence 1 (chat): an insult"), decided.GetComment())

	moderation, err := chatv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL).ListModeration(t.Context(),
		connect.NewRequest(&chatv1.ListModerationRequest{AccountId: raider.account()}))
	require.NoError(t, err)
	require.Len(t, moderation.Msg.GetActions(), 1, "a first offence removes, and mutes nobody")
	action := moderation.Msg.GetActions()[0]
	assert.Equal(t, said.GetId(), action.GetRemoval().GetMessageId())
	assert.Equal(t, "Claude judge", action.GetClient())
	assert.Equal(t, "offence 1 (chat): an insult", action.GetReason())

	for _, message := range reporter.history().GetMessages() {
		if message.GetId() == said.GetId() {
			assert.True(t, message.GetRemoved(), "everybody reads it removed")
		}
	}
}
