package get_freeze_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_freeze_handler"
)

func TestTheAnswerIsTheFreezeNow(t *testing.T) {
	frozen := clicks.Freeze(false)
	handler := get_freeze_handler.New(func() clicks.Freeze { return frozen })

	for _, now := range []bool{false, true} {
		frozen = clicks.Freeze(now)

		res, err := handler.GetFreeze(t.Context(), connect.NewRequest(&planetv1.GetFreezeRequest{}))
		require.NoError(t, err)
		assert.Equal(t, now, res.Msg.GetFrozen())
	}
}
