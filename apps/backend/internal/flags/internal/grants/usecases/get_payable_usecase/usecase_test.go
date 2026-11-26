package get_payable_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/get_payable_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var (
	now     = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	founder = catalog.AccountID{15: 1}
	other   = catalog.AccountID{15: 2}
	id      = submissions.SubmissionID{15: 1}
)

func seeded(t *testing.T, state submissions.State, flag catalog.Flag) *get_payable_usecase.UseCase {
	t.Helper()
	return seededPaid(t, state, flag, false)
}

func seededPaid(t *testing.T, state submissions.State, flag catalog.Flag, paid bool) *get_payable_usecase.UseCase {
	t.Helper()
	kept, flags := submissions.NewMemoryStore(), catalog.NewMemoryStore()
	row := submissions.TestWaiting(id, founder, now.Add(-time.Hour))
	row.State, row.Held, row.Paid = state, state == submissions.Waiting || state == submissions.Rejected, paid
	if state == submissions.Rejected {
		row.Rejection = submissions.Rejection{Reason: submissions.Hateful}
	}
	if state == submissions.Opened {
		row.Flag = flag.TakeSnapshot().ID
		require.NoError(t, flags.Save(t.Context(), flag))
	}
	kept.Seed(row)
	return get_payable_usecase.New(kept, flags)
}

func target(t *testing.T, submission string, flag catalog.FlagID) grants.Target {
	t.Helper()
	parsed, err := grants.ParseTarget(submission, flag.String())
	require.NoError(t, err)
	return parsed
}

func payable(t *testing.T, useCase *get_payable_usecase.UseCase, account catalog.AccountID, target grants.Target) error {
	t.Helper()
	_, err := useCase.Execute(t.Context(), get_payable_usecase.In{Account: account.String(), Target: target})
	return err
}

func TestTheFounderPaysForItsApprovedSubmission(t *testing.T) {
	useCase := seeded(t, submissions.Approved, catalog.Flag{})

	same, err := useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: founder.String(), Target: target(t, id.String(), "")})
	require.NoError(t, err)
	assert.Equal(t, []grants.Target{target(t, id.String(), "")}, same, "no flag yet")
}

func TestAnotherAccountDoesNotPayForASubmission(t *testing.T) {
	useCase := seeded(t, submissions.Approved, catalog.Flag{})

	require.ErrorIs(t, payable(t, useCase, other, target(t, id.String(), "")), submissions.ErrNotYours)
}

func TestTheFounderPaysForASubmissionThatWaitsOnce(t *testing.T) {
	require.NoError(t, payable(t, seeded(t, submissions.Waiting, catalog.Flag{}), founder, target(t, id.String(), "")))

	err := payable(t, seededPaid(t, submissions.Waiting, catalog.Flag{}, true), founder, target(t, id.String(), ""))

	require.ErrorIs(t, err, submissions.ErrPaidAlready)
}

func TestNobodyPaysForASubmissionRejected(t *testing.T) {
	err := payable(t, seeded(t, submissions.Rejected, catalog.Flag{}), founder, target(t, id.String(), ""))

	require.ErrorIs(t, err, submissions.ErrNotPayable)
}

func TestTheHolderPaysForItsFlagAndBothTargetsBuyTimeForIt(t *testing.T) {
	useCase := seeded(t, submissions.Opened, catalog.HeldFlag("r-0000000001", "Rennes", founder, now))

	same, err := useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: founder.String(), Target: target(t, "", "r-0000000001")})
	require.NoError(t, err)
	assert.Equal(t, []grants.Target{target(t, "", "r-0000000001"), target(t, id.String(), "")}, same)

	same, err = useCase.Execute(t.Context(),
		get_payable_usecase.In{Account: founder.String(), Target: target(t, id.String(), "")})
	require.NoError(t, err)
	assert.Equal(t, []grants.Target{target(t, id.String(), ""), target(t, "", "r-0000000001")}, same)
}

func TestAnyAccountPaysForAFlagWhoseTimeEndedHoweverLongAgo(t *testing.T) {
	ended, ok := catalog.HeldFlag("r-0000000001", "Rennes", founder, now.AddDate(-2, 0, 0)).Expired(now)
	require.True(t, ok)
	useCase := seeded(t, submissions.Opened, ended)

	require.NoError(t, payable(t, useCase, other, target(t, "", "r-0000000001")), "a claim")
	require.NoError(t, payable(t, useCase, founder, target(t, "", "r-0000000001")))
}

func TestNobodyPaysForARemovedFlag(t *testing.T) {
	removed, err := catalog.HeldFlag("r-0000000001", "Rennes", founder, now.Add(-time.Hour)).Removed("Broke the rules.", now)
	require.NoError(t, err)
	useCase := seeded(t, submissions.Opened, removed)

	require.ErrorIs(t, payable(t, useCase, founder, target(t, "", "r-0000000001")), catalog.ErrRemoved)
	require.ErrorIs(t, payable(t, useCase, founder, target(t, id.String(), "")), catalog.ErrRemoved)
	require.ErrorIs(t, payable(t, useCase, other, target(t, "", "r-0000000001")), catalog.ErrRemoved)
}

func TestAnotherAccountDoesNotPayForAnOpenFlag(t *testing.T) {
	useCase := seeded(t, submissions.Opened, catalog.HeldFlag("r-0000000001", "Rennes", founder, now))

	require.ErrorIs(t, payable(t, useCase, other, target(t, "", "r-0000000001")), catalog.ErrHeld)
}

func TestTheFounderNoLongerPaysForItsFlagOnceAnotherAccountClaimedIt(t *testing.T) {
	ended, ok := catalog.HeldFlag("r-0000000001", "Rennes", founder, now.AddDate(0, -2, 0)).Expired(now.Add(-time.Hour))
	require.True(t, ok)
	claimed, _, err := ended.Paid(other, now, catalog.Days(7), now)
	require.NoError(t, err)
	useCase := seeded(t, submissions.Opened, claimed)

	require.ErrorIs(t, payable(t, useCase, founder, target(t, "", "r-0000000001")), catalog.ErrHeld)
	require.ErrorIs(t, payable(t, useCase, founder, target(t, id.String(), "")), catalog.ErrHeld,
		"through its submission neither")
	require.NoError(t, payable(t, useCase, other, target(t, "", "r-0000000001")))
}
