package get_attachment_rules_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/get_attachment_rules_handler"
)

type room struct {
	bytes int64
	err   error
}

func (r room) Room(context.Context) (int64, error) {
	return r.bytes, r.err
}

const plenty = 1 << 40

func rulesOf(t *testing.T, rules notes.AttachmentRules, left room) *connect.Response[supportv1.GetAttachmentRulesResponse] {
	t.Helper()

	res, err := get_attachment_rules_handler.New(rules, left).GetAttachmentRules(t.Context(),
		connect.NewRequest(&supportv1.GetAttachmentRulesRequest{}))
	require.NoError(t, err)
	return res
}

func TestTheRulesAreAnsweredWithEveryTypeANoteCarriesAndAreNotCached(t *testing.T) {
	res := rulesOf(t, notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 50 << 20}, room{bytes: plenty})

	assert.Equal(t, uint32(10), res.Msg.GetMaxPerNote())
	assert.Equal(t, uint64(50<<20), res.Msg.GetMaxBytes())
	assert.Len(t, res.Msg.GetContentTypes(), len(notes.MediaTypes()))
	assert.Contains(t, res.Msg.GetContentTypes(), "image/png")
	assert.Contains(t, res.Msg.GetContentTypes(), "video/mp4")
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAServerThatTakesNoAttachmentSaysZero(t *testing.T) {
	res := rulesOf(t, notes.AttachmentRules{}, room{bytes: plenty})

	assert.Zero(t, res.Msg.GetMaxPerNote())
	assert.Zero(t, res.Msg.GetMaxBytes())
}

func TestTheRulesShrinkToTheRoomTheBucketHasLeftAndAFullOneTakesNoFile(t *testing.T) {
	rules := notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 50 << 20}

	low := rulesOf(t, rules, room{bytes: 1000})
	full := rulesOf(t, rules, room{})

	assert.Equal(t, uint32(10), low.Msg.GetMaxPerNote())
	assert.Equal(t, uint64(1000), low.Msg.GetMaxBytes())
	assert.Zero(t, full.Msg.GetMaxPerNote())
	assert.Zero(t, full.Msg.GetMaxBytes())
}

func TestARoomThatCannotBeReadIsLeftToTheErrorNet(t *testing.T) {
	broken := errors.New("the database is gone")

	_, err := get_attachment_rules_handler.New(notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 1000}, room{err: broken}).
		GetAttachmentRules(t.Context(), connect.NewRequest(&supportv1.GetAttachmentRulesRequest{}))

	require.ErrorIs(t, err, broken)
}
