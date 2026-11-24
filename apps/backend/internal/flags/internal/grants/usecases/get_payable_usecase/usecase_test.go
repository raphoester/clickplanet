package get_payable_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/get_payable_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	founder = submissions.AccountID{15: 1}
	other   = submissions.AccountID{15: 2}
	id      = submissions.SubmissionID{15: 1}
)

func seeded(t *testing.T, state submissions.State, flag catalog.Flag) *get_payable_usecase.UseCase {
	t.Helper()
	kept, flags := submissions.NewMemoryStore(), catalog.NewMemoryStore()
	row := submissions.TestWaiting(id, founder, now.Add(-time.Hour))
	row.State, row.Held = state, state == submissions.Waiting
	if state == submissions.Opened {
		row.Flag = flag.TakeSnapshot().ID
		require.NoError(t, flags.Save(t.Context(), flag))
	}
	kept.Seed(row)
	return get_payable_usecase.New(cptime.NewFixedClock(now), kept, flags)
}

func target(t *testing.T, submission string, flag catalog.FlagID) grants.Target {
	t.Helper()
	parsed, err := grants.ParseTarget(submission, flag.String())
	require.NoError(t, err)
	return parsed
}

func TestTheFounderPaysForItsApprovedSubmission(t *testing.T) {
	useCase := seeded(t, submissions.Approved, catalog.Flag{})

	require.NoError(t, useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: founder.String(), Target: target(t, id.String(), "")}))
}

func TestAnotherAccountDoesNotPayForASubmission(t *testing.T) {
	useCase := seeded(t, submissions.Approved, catalog.Flag{})

	err := useCase.Execute(t.Context(), get_payable_usecase.In{Account: other.String(), Target: target(t, id.String(), "")})

	require.ErrorIs(t, err, submissions.ErrNotYours)
}

func TestNobodyPaysForASubmissionThatWaits(t *testing.T) {
	useCase := seeded(t, submissions.Waiting, catalog.Flag{})

	err := useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: founder.String(), Target: target(t, id.String(), "")})

	require.ErrorIs(t, err, submissions.ErrNotPayable)
}

func TestTheFounderPaysForItsFlagWhileItMayComeBack(t *testing.T) {
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", now.Add(-40*24*time.Hour)).Closed("Broke the rules.", now.Add(-29*24*time.Hour))
	require.NoError(t, err)
	useCase := seeded(t, submissions.Opened, closed)

	require.NoError(t, useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: founder.String(), Target: target(t, "", "r-0000000001")}))
	require.NoError(t, useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: founder.String(), Target: target(t, id.String(), "")}))
}

func TestNobodyPaysForAFlagClosedThirtyDaysAgo(t *testing.T) {
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", now.Add(-90*24*time.Hour)).Closed("Broke the rules.", now.Add(-30*24*time.Hour))
	require.NoError(t, err)
	useCase := seeded(t, submissions.Opened, closed)

	err = useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: founder.String(), Target: target(t, "", "r-0000000001")})

	require.ErrorIs(t, err, catalog.ErrTooLate)
}

func TestAnotherAccountDoesNotPayForAFlag(t *testing.T) {
	useCase := seeded(t, submissions.Opened, catalog.OpenFlag("r-0000000001", "Rennes", now))

	err := useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: other.String(), Target: target(t, "", "r-0000000001")})

	require.ErrorIs(t, err, submissions.ErrNotYours)
}
