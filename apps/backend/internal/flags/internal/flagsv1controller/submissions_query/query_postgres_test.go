package submissions_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissions_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/postgres_submission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

type links struct{}

func (links) HeldURL(key string, _ time.Duration) (string, error) {
	return "https://bucket.test/" + key, nil
}

var (
	at  = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada = submissions.AccountID{15: 1}
	bob = submissions.AccountID{15: 2}
)

func ids[T interface{ GetId() string }](messages []T) []string {
	read := make([]string, 0, len(messages))
	for _, message := range messages {
		read = append(read, message.GetId()[34:])
	}
	return read
}

func TestTheOperatorReadsTheWaitingOldestFirstThenTheDecidedLatestFirst(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	store := postgres_submission_store.New(db)
	query := submissions_query.NewPostgresQuery(db,
		submissionmessage.New(flagmessage.New("https://media.test"), links{}))

	late := submissions.TestWaiting(submissions.SubmissionID{15: 1}, ada, at.Add(2*time.Hour))
	early := submissions.TestWaiting(submissions.SubmissionID{15: 2}, bob, at)
	rejected := submissions.TestWaiting(submissions.SubmissionID{15: 3}, ada, at.Add(-48*time.Hour))
	rejected.State, rejected.DecidedAt = submissions.Rejected, at.Add(-24*time.Hour)
	rejected.Rejection = submissions.Rejection{Reason: submissions.Other, Note: "A city."}
	approved := submissions.TestWaiting(submissions.SubmissionID{15: 4}, bob, at.Add(-72*time.Hour))
	approved.State, approved.DecidedAt, approved.Held = submissions.Approved, at.Add(-time.Hour), false
	for _, row := range []submissions.Snapshot{late, early, rejected, approved} {
		require.NoError(t, store.Save(t.Context(), submissions.Restored(row)))
	}

	kept, err := query.Kept(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"02", "01", "04", "03"}, ids(kept.GetSubmissions()))
	assert.Equal(t, "https://bucket.test/"+early.ID.PictureKey(), kept.GetSubmissions()[0].GetPicture().GetUrl())
	assert.Equal(t, "A city.", kept.GetSubmissions()[3].GetRejection().GetNote())

	mine, err := query.Mine(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, []string{"01", "03"}, ids(mine.GetSubmissions()), "the newest first")
	assert.Equal(t, flagsv1.SubmissionState_SUBMISSION_STATE_REJECTED, mine.GetSubmissions()[1].GetState())
}

func TestAFounderReadsWhyTheOperatorClosedItsFlagsAndNoOtherFlag(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	store, flags := postgres_submission_store.New(db), postgres_flag_store.New(db)
	query := submissions_query.NewPostgresQuery(db,
		submissionmessage.New(flagmessage.New("https://media.test"), links{}))

	founded := func(flag catalog.FlagID, name catalog.Name, account submissions.AccountID, id byte, note catalog.Note) {
		t.Helper()
		opened := catalog.OpenFlag(flag, name, at)
		if note != "" {
			closed, err := opened.Closed(note, at.Add(time.Hour))
			require.NoError(t, err)
			opened = closed
		}
		require.NoError(t, flags.Save(t.Context(), opened))
		row := submissions.TestWaiting(submissions.SubmissionID{15: id}, account, at.Add(-time.Hour))
		row.State, row.Flag, row.Name, row.Held = submissions.Opened, flag, name, false
		require.NoError(t, store.Save(t.Context(), submissions.Restored(row)))
	}
	founded("r-0000000001", "Bavaria", ada, 1, "Broke the rules.")
	founded("r-0000000002", "Swabia", ada, 2, "")
	founded("r-0000000003", "Tyrol", bob, 3, "A test.")
	change := submissions.TestWaiting(submissions.SubmissionID{15: 4}, ada, at)
	change.Kind, change.Flag, change.State, change.Held = submissions.Change, "r-0000000003", submissions.Approved, false
	require.NoError(t, store.Save(t.Context(), submissions.Restored(change)))

	mine, err := query.Mine(t.Context(), ada)
	require.NoError(t, err)
	require.Len(t, mine.GetClosures(), 1, "not a flag open, nor one another account founded")
	assert.Equal(t, "r-0000000001", mine.GetClosures()[0].GetFlag())
	assert.Equal(t, "Broke the rules.", mine.GetClosures()[0].GetNote())
}
