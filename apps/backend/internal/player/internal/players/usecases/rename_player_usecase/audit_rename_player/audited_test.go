package audit_rename_player_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/rename_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/rename_player_usecase/audit_rename_player"
)

var ada = players.AccountID{15: 1}

type stubUseCase struct {
	err error
}

func (s stubUseCase) Execute(context.Context, rename_player_usecase.In) (players.Profile, error) {
	return players.Profile{}, s.err
}

func run(t *testing.T, inner stubUseCase) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_rename_player.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), rename_player_usecase.In{Account: ada, Name: "Ada_L"})
	return logs.String(), err
}

func TestARenameIsLoggedWithTheAccountAndTheName(t *testing.T) {
	logs, err := run(t, stubUseCase{})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin player rename" account=`+ada.String()+` name=Ada_L`)
}

func TestAFailedRenameIsLoggedToo(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin player rename failed" account=`+ada.String()+` name=Ada_L`)
}
