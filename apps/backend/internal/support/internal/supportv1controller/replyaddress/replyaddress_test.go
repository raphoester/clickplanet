package replyaddress_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/replyaddress"
)

func sent(address string) *string {
	return &address
}

func TestAnAddressSentIsReadAndNoneIsEmpty(t *testing.T) {
	address, err := replyaddress.Of(sent(" Ada@Example.com "))
	require.NoError(t, err)
	assert.Equal(t, cpmail.Address("ada@example.com"), address)

	none, err := replyaddress.Of(nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestAnAddressMailCannotGoToIsRefusedWithADetailThePageReads(t *testing.T) {
	_, err := replyaddress.Of(sent("ada@localhost"))

	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	assert.Equal(t, connect.CodeInvalidArgument, refused.Code())
	require.Len(t, refused.Details(), 1)
	detail, err := refused.Details()[0].Value()
	require.NoError(t, err)
	assert.Equal(t, supportv1.NoteRefusalReason_NOTE_REFUSAL_REASON_INVALID_REPLY_ADDRESS, detail.(*supportv1.NoteRefusal).GetReason())
}
