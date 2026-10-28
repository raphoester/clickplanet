package account_deleted_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/account_deleted_subscriber"
)

func TestADeletedAccountLeavesItsMissions(t *testing.T) {
	store := inmemory_mission_store.New()
	account, err := missions.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	day := missions.NewCalendar(missions.Config{}).DayOf(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.ChangeSheet(t.Context(), day, account, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Progress: map[missions.MissionID]uint32{{15: 1}: 1}})
	}))

	err = account_deleted_subscriber.New(forget_account_usecase.New(store)).Handle(t.Context(),
		&authv1.AccountDeleted{AccountId: account.String()})

	require.NoError(t, err)
	sheet, err := store.Sheet(t.Context(), day, account)
	require.NoError(t, err)
	assert.Empty(t, sheet.Snapshot().Progress)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := account_deleted_subscriber.New(forget_account_usecase.New(inmemory_mission_store.New())).
		Handle(t.Context(), &authv1.AccountDeleted{})

	assert.ErrorIs(t, err, missions.ErrInvalidAccount)
}
