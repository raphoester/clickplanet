package anonymize_recruitments_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_recruitment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/usecases/anonymize_recruitments_usecase"
)

var (
	ada = arrivals.AccountID{15: 1}
	bob = arrivals.AccountID{15: 2}
)

func TestTheRecruitersRecruitmentsStayAndLoseTheRecruiter(t *testing.T) {
	store := inmemory_recruitment_store.New()
	kept := arrivals.RecruitmentSnapshot{Arrival: arrivals.ArrivalID{15: 3}, Recruiter: bob}
	store.Seed(
		arrivals.RecruitmentSnapshot{Arrival: arrivals.ArrivalID{15: 1}, Recruiter: ada},
		kept,
		arrivals.RecruitmentSnapshot{Arrival: arrivals.ArrivalID{15: 2}, Recruiter: ada},
	)

	require.NoError(t, anonymize_recruitments_usecase.New(store).Execute(t.Context(), ada))

	assert.ElementsMatch(t, []arrivals.RecruitmentSnapshot{
		{Arrival: arrivals.ArrivalID{15: 1}},
		kept,
		{Arrival: arrivals.ArrivalID{15: 2}},
	}, store.Kept())
}

func TestAnAccountThatRecruitedNobodyChangesNothing(t *testing.T) {
	store := inmemory_recruitment_store.New()
	kept := arrivals.RecruitmentSnapshot{Arrival: arrivals.ArrivalID{15: 1}, Recruiter: bob}
	store.Seed(kept)

	require.NoError(t, anonymize_recruitments_usecase.New(store).Execute(t.Context(), ada))

	assert.Equal(t, []arrivals.RecruitmentSnapshot{kept}, store.Kept())
}

func TestAFailureToAnonymizeIsAnError(t *testing.T) {
	store := inmemory_recruitment_store.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	assert.ErrorIs(t, anonymize_recruitments_usecase.New(store).Execute(t.Context(), ada), refused)
}
