package get_attachment_rules_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/get_attachment_rules_handler"
)

func rulesOf(t *testing.T, rules notes.AttachmentRules) *connect.Response[supportv1.GetAttachmentRulesResponse] {
	t.Helper()

	res, err := get_attachment_rules_handler.New(rules).GetAttachmentRules(t.Context(),
		connect.NewRequest(&supportv1.GetAttachmentRulesRequest{}))
	require.NoError(t, err)
	return res
}

func TestTheRulesAreAnsweredWithEveryTypeANoteCarriesAndAreNotCached(t *testing.T) {
	res := rulesOf(t, notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 50 << 20})

	assert.Equal(t, uint32(10), res.Msg.GetMaxPerNote())
	assert.Equal(t, uint64(50<<20), res.Msg.GetMaxBytes())
	assert.Len(t, res.Msg.GetContentTypes(), len(notes.MediaTypes()))
	assert.Contains(t, res.Msg.GetContentTypes(), "image/png")
	assert.Contains(t, res.Msg.GetContentTypes(), "video/mp4")
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAServerThatTakesNoAttachmentSaysZero(t *testing.T) {
	res := rulesOf(t, notes.AttachmentRules{})

	assert.Zero(t, res.Msg.GetMaxPerNote())
	assert.Zero(t, res.Msg.GetMaxBytes())
}
