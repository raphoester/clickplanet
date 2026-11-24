//go:build testing

package submissions

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

// TestWaiting is a new flag of Rennes the account submitted at a time, waiting, with the test picture.
func TestWaiting(id SubmissionID, account AccountID, at time.Time) Snapshot {
	return Snapshot{
		ID: id, Account: account, Kind: New, Name: "Rennes", Pin: catalog.TestPin, Picture: catalog.TestPicture,
		OtherColors: 3, Held: true, State: Waiting, SubmittedAt: at, DecidedAt: at,
	}
}

// TestDrawn is the test picture, drawn again.
var TestDrawn = func() pictures.Drawn {
	picture, err := catalog.TestPicture.Restore()
	if err != nil {
		panic(err)
	}
	return pictures.Drawn{PNG: []byte("png"), Picture: picture, OtherColors: 3}
}()

func Restored(snapshot Snapshot) Submission {
	submission, err := snapshot.Restore()
	if err != nil {
		panic(err)
	}
	return submission
}
