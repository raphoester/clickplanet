package account_deleted_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_arrival_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_recruitment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/usecases/anonymize_arrivals_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/usecases/anonymize_recruitments_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/subscribers/account_deleted_subscriber"
)

var at = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestADeletedAccountsArrivalLosesItsAccount(t *testing.T) {
	store := inmemory_arrival_store.New()
	ada := arrivals.AccountID{15: 1}
	store.Seed(arrivals.Snapshot{ID: arrivals.ArrivalID{15: 1}, Account: ada, Source: arrivals.SourceRecruit, Referrer: "discord.com", At: at})

	err := account_deleted_subscriber.New(anonymize_arrivals_usecase.New(store)).Handle(t.Context(),
		&authv1.AccountDeleted{AccountId: ada.String()})

	require.NoError(t, err)
	assert.Equal(t, []arrivals.Snapshot{{ID: arrivals.ArrivalID{15: 1}, Source: arrivals.SourceRecruit, Referrer: "discord.com", At: at}},
		store.Kept())
}

func TestARecruitAndItsRecruiterAreEachForgottenWhenTheirAccountGoes(t *testing.T) {
	recruit, recruiter := arrivals.AccountID{15: 1}, arrivals.AccountID{15: 2}
	arrival := arrivals.ArrivalID{15: 1}
	kept := inmemory_arrival_store.New()
	kept.Seed(arrivals.Snapshot{ID: arrival, Account: recruit, Source: arrivals.SourceRecruit, At: at})
	recruitments := inmemory_recruitment_store.New()
	recruitments.Seed(arrivals.RecruitmentSnapshot{Arrival: arrival, Recruiter: recruiter})
	deleted := func(account arrivals.AccountID) {
		event := &authv1.AccountDeleted{AccountId: account.String()}
		require.NoError(t, account_deleted_subscriber.New(anonymize_arrivals_usecase.New(kept)).Handle(t.Context(), event))
		require.NoError(t, account_deleted_subscriber.New(anonymize_recruitments_usecase.New(recruitments)).Handle(t.Context(), event))
	}

	deleted(recruit)

	assert.Equal(t, []arrivals.Snapshot{{ID: arrival, Source: arrivals.SourceRecruit, At: at}}, kept.Kept())
	assert.Equal(t, []arrivals.RecruitmentSnapshot{{Arrival: arrival, Recruiter: recruiter}}, recruitments.Kept(),
		"the recruitment names the recruit only through its arrival, which names nobody now")

	deleted(recruiter)

	assert.Equal(t, []arrivals.RecruitmentSnapshot{{Arrival: arrival}}, recruitments.Kept())
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := account_deleted_subscriber.New(anonymize_arrivals_usecase.New(inmemory_arrival_store.New())).Handle(t.Context(),
		&authv1.AccountDeleted{})

	require.ErrorIs(t, err, accounts.ErrInvalidAccount)
}
