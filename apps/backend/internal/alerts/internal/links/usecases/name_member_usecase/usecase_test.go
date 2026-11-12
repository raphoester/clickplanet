package name_member_usecase_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/name_member_usecase"
)

var (
	now     = time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
)

func TestTheLinkedUserWearsTheAccountsUsername(t *testing.T) {
	store, names, members := links.NewInMemoryStore(), links.NewFakeNames(), &links.FakeRenamer{}
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "42", "ie", now)))
	names.Name(account, "BraveFox42")

	require.NoError(t, name_member_usecase.New(links.NewNamer(store, names, members)).Execute(t.Context(), account))

	assert.Equal(t, []string{"42 BraveFox42"}, members.Renamed())
}
