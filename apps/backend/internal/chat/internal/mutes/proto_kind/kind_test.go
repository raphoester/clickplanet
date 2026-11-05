package proto_kind_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/proto_kind"
)

func TestEveryKindCrossesTheWireBothWays(t *testing.T) {
	for _, kind := range []mutes.Kind{mutes.Chat, mutes.Gifs} {
		back, err := proto_kind.Decode(proto_kind.Encode(kind))
		require.NoError(t, err)
		assert.Equal(t, kind, back)
	}
	for number := range chatv1.MuteKind_name {
		_, err := proto_kind.Decode(chatv1.MuteKind(number))
		require.NoError(t, err, "every kind the proto names is one here: %d", number)
	}
}

func TestAKindTheProtoDoesNotNameIsInvalid(t *testing.T) {
	_, err := proto_kind.Decode(chatv1.MuteKind(7))

	assert.ErrorIs(t, err, mutes.ErrInvalidKind)
}
