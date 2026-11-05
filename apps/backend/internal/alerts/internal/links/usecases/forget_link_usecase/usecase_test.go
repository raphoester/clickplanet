package forget_link_usecase_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/forget_link_usecase"
)

func TestADeletedAccountLeavesNoLink(t *testing.T) {
	now := time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	account := links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
	store := links.NewInMemoryStore()
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "ie", now)))
	useCase := forget_link_usecase.New(store)

	require.NoError(t, useCase.Execute(t.Context(), account))
	require.NoError(t, useCase.Execute(t.Context(), account), "a second delivery finds nothing")

	assert.Empty(t, store.Links())
}
