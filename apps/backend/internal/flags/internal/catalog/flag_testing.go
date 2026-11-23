//go:build testing

package catalog

import (
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

// TestPicture is a picture drawn again, as a flag opened in a test wears.
var TestPicture = pictures.Snapshot{
	Hash: strings.Repeat("ab", 32), Width: 300, Height: 200, Fit: pictures.Fit{FocusX: 0.5, FocusY: 0.5},
}

// OpenFlag is a flag opened at a time, with 30 days to run and the test picture.
func OpenFlag(id FlagID, name Name, at time.Time) Flag {
	flag, err := Snapshot{
		ID: id, Name: name, Picture: TestPicture, State: Open, DecidedAt: at, EndsAt: at.Add(30 * 24 * time.Hour),
	}.Restore()
	if err != nil {
		panic(err)
	}
	return flag
}
