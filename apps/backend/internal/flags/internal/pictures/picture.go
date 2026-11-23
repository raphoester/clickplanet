package pictures

import (
	"errors"
	"fmt"
	"regexp"
)

var ErrInvalidHash = errors.New("not the name of a picture drawn again")

var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Picture is a flag's picture as the server drew it again, named after its content: public, and kept a year by a
// browser, since a name never holds another picture.
type Picture struct {
	hash   string
	width  uint32
	height uint32
	fit    Fit
}

// Key is where the bucket keeps it.
func (p Picture) Key() string {
	return p.TakeSnapshot().Key()
}

func (p Picture) TakeSnapshot() Snapshot {
	return Snapshot{Hash: p.hash, Width: p.width, Height: p.height, Fit: p.fit}
}

type Snapshot struct {
	Hash   string
	Width  uint32
	Height uint32
	Fit    Fit
}

func (s Snapshot) Restore() (Picture, error) {
	if !hashPattern.MatchString(s.Hash) {
		return Picture{}, fmt.Errorf("%w: %q", ErrInvalidHash, s.Hash)
	}
	return Picture{hash: s.Hash, width: s.Width, height: s.Height, fit: s.Fit}, nil
}

// Key is where the bucket keeps the picture of this snapshot.
func (s Snapshot) Key() string {
	return "flags/pictures/" + s.Hash + ".png"
}
