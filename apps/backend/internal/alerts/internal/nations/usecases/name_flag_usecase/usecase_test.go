package name_flag_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations/usecases/name_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

var at = time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)

func TestARenameRenamesTheRoleOnce(t *testing.T) {
	catalog, err := nations.Load(t.Context(), nations.NewInMemoryStore())
	require.NoError(t, err)
	maker := &roles.FakeMaker{}
	book := roles.NewBook(roles.NewInMemoryStore(), maker, catalog)
	useCase := name_flag_usecase.New(catalog, book)

	require.NoError(t, useCase.Execute(t.Context(), name_flag_usecase.In{Code: "r-jz9cwqzp0k", Name: "Holy Roman Empire", At: at}))
	flag, ok := catalog.FlagOf("r-jz9cwqzp0k")
	require.True(t, ok, "a sold flag named has a role to make")
	id, err := book.Ensure(t.Context(), roles.Nation(flag))
	require.NoError(t, err)

	rename := name_flag_usecase.In{Code: "r-jz9cwqzp0k", Name: "Heiliges Römisches Reich", At: at.Add(time.Hour)}
	require.NoError(t, useCase.Execute(t.Context(), rename))
	require.NoError(t, useCase.Execute(t.Context(), rename), "told again")

	assert.Equal(t, map[discord.RoleID]string{id: "Heiliges Römisches Reich"}, maker.Renamed)
	assert.Equal(t, []string{"Holy Roman Empire"}, maker.Made)
}
