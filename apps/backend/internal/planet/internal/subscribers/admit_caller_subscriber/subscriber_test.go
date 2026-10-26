package admit_caller_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/admit_caller_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/refuse_caller_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

var at = time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)

func TestAnAdmissionThatArrivesBeforeTheRefusalItEndsStillEndsIt(t *testing.T) {
	clock := cptime.NewFixedClock(at)
	book := refusals.NewBook(refusals.NewMemoryStore(), clock)
	admitted := &planetv1.AdmitCaller{
		Caller:    &planetv1.AdmitCaller_AccountId{AccountId: account},
		DecidedAt: timestamppb.New(at.Add(time.Minute)),
	}

	require.NoError(t, admit_caller_subscriber.New(book).Handle(t.Context(), admitted))
	require.NoError(t, refuse_caller_subscriber.New(book).Handle(t.Context(), &planetv1.RefuseCaller{
		Caller:    &planetv1.RefuseCaller_AccountId{AccountId: account},
		Until:     timestamppb.New(at.Add(24 * time.Hour)),
		DecidedAt: timestamppb.New(at),
	}))
	require.NoError(t, admit_caller_subscriber.New(book).Handle(t.Context(), admitted))

	clock.Advance(2 * time.Minute)
	_, refused, err := book.Refusal(t.Context(), clicks.Payer{Scope: "203.0.113.7", Account: account})
	require.NoError(t, err)
	assert.False(t, refused, "the two commands travel on two streams: the later decision wins, not the later delivery")
}

func TestAnAdmissionThatNamesAMalformedAccountIsRefused(t *testing.T) {
	err := admit_caller_subscriber.New(refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(at))).
		Handle(t.Context(), &planetv1.AdmitCaller{Caller: &planetv1.AdmitCaller_AccountId{AccountId: "nope"}})

	assert.ErrorIs(t, err, refusals.ErrInvalidAccount)
}
