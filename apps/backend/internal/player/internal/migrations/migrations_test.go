package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestEveryEventPlayerTellsHasAPlaceInItsOutbox(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "player", migrations.FS)
	account := "00000000-0000-0000-0000-000000000001"

	for _, event := range []proto.Message{
		&playerv1.PlayerCreated{AccountId: account},
		&playerv1.PlayerSignedIn{AccountId: account},
		&playerv1.PlayerNamed{AccountId: account},
		&playerv1.PlayerLeaving{AccountId: account},
		&playerv1.PlayerReturned{AccountId: account},
		&playerv1.PlayerForgotten{AccountId: account},
	} {
		assert.NoError(t, db.InTx(t.Context(), func(ctx context.Context) error {
			return cpoutbox.New(db).Record(ctx, event)
		}), "%s", event.ProtoReflect().Descriptor().FullName())
	}
}
