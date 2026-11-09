package post_teasers_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/post_teasers_usecase"
)

type renders []teaser_job.Rendered

func (r renders) Rendered(context.Context) ([]teaser_job.Rendered, error) {
	return r, nil
}

// Uploads a file under its own name, busy as many times as told first.
type storage struct {
	busy     int
	uploaded []string
}

func (s *storage) Upload(_ context.Context, file teaser_job.File, contentType string) (teaser_job.MediaURL, error) {
	if s.busy > 0 {
		s.busy--
		return "", &teaser_job.BusyError{Reason: "the storage answered 503"}
	}
	s.uploaded = append(s.uploaded, string(file)+" "+contentType)
	return teaser_job.MediaURL("https://media.example/" + string(file)), nil
}

// Answers each of answers in turn.
type promo struct {
	answers []error
	posted  []teaser_job.Media
}

func (p *promo) Post(_ context.Context, battle teaser_job.BattleID, media teaser_job.Media) (teaser_job.Outcome, error) {
	p.posted = append(p.posted, media)
	err := p.answers[0]
	p.answers = p.answers[1:]
	if err != nil {
		return teaser_job.Outcome{}, err
	}
	return teaser_job.Outcome{Battle: battle, Result: teaser_job.Posted, Post: "post-1"}, nil
}

type pauses []time.Duration

func (p *pauses) Wait(_ context.Context, d time.Duration) error {
	*p = append(*p, d)
	return nil
}

var (
	backoff = teaser_job.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}
	ring    = teaser_job.Rendered{Battle: "entry-7", Photos: []teaser_job.File{"card-1.jpg", "card-2.jpg"}, Video: "teaser.mp4"}
)

func execute(t *testing.T, store *storage, poster *promo, rendered ...teaser_job.Rendered) (teaser_job.Report, pauses) {
	t.Helper()
	waited := pauses{}
	report, err := post_teasers_usecase.New(renders(rendered), store, poster, backoff, &waited).Execute(t.Context())
	require.NoError(t, err)
	return report, waited
}

func TestEachRenderIsUploadedThenHandedToPromo(t *testing.T) {
	store, poster := &storage{}, &promo{answers: []error{nil}}

	report, waited := execute(t, store, poster, ring)

	assert.Equal(t, teaser_job.Report{{Battle: "entry-7", Result: teaser_job.Posted, Post: "post-1"}}, report)
	assert.Equal(t, []string{"card-1.jpg image/jpeg", "card-2.jpg image/jpeg", "teaser.mp4 video/mp4"}, store.uploaded)
	assert.Equal(t, []teaser_job.Media{{
		Photos: []teaser_job.MediaURL{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"},
		Video:  "https://media.example/teaser.mp4",
	}}, poster.posted)
	assert.Empty(t, waited)
}

func TestABusyStorageOrPromoIsWaitedForThenTriedAgain(t *testing.T) {
	busy := &teaser_job.BusyError{RetryAfter: 2 * time.Minute, Reason: "zernio answered 429"}

	report, waited := execute(t, &storage{busy: 1}, &promo{answers: []error{busy, nil}}, ring)

	assert.Equal(t, teaser_job.Posted, report[0].Result)
	assert.Equal(t, pauses{time.Minute, 2 * time.Minute}, waited)
}

func TestAPromoBusyPastTheBackoffOrARefusalFailsThatTeaserAlone(t *testing.T) {
	busy := &teaser_job.BusyError{Reason: "publishing is restarting"}
	other := teaser_job.Rendered{Battle: "entry-8", Photos: []teaser_job.File{"card-1.jpg"}, Video: "teaser.mp4"}

	report, waited := execute(t, &storage{}, &promo{answers: []error{busy, busy, busy, busy, errors.New("the backend refused it"), nil}}, ring, other, ring)

	assert.Equal(t, []teaser_job.Result{teaser_job.Failed, teaser_job.Failed, teaser_job.Posted}, []teaser_job.Result{report[0].Result, report[1].Result, report[2].Result})
	assert.Equal(t, "publishing is restarting", report[0].Why)
	assert.Equal(t, "the backend refused it", report[1].Why)
	assert.Equal(t, teaser_job.BattleID("entry-8"), report[1].Battle)
	assert.Equal(t, pauses(backoff), waited)
	assert.True(t, report.Failed())
}
