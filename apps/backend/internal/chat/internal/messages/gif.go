package messages

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// KLIPY's media hosts alone: a GIF never makes a reader's browser load an address a player chose.
var gifHosts = cpcolls.NewSet("static.klipy.com", "static1.klipy.com", "static2.klipy.com")

const (
	maxGifURLLength = 512
	maxGifSide      = 2000
)

var ErrGuestGif = errors.New("a guest may not post a GIF")

// At the address KLIPY gave: its terms forbid rewriting it.
type Gif struct {
	URL    string
	Width  uint32
	Height uint32
}

func (g Gif) None() bool { return g == Gif{} }

func (g Gif) checked() (Gif, error) {
	if len(g.URL) > maxGifURLLength {
		return Gif{}, fmt.Errorf("an address longer than %d bytes", maxGifURLLength)
	}
	address, err := url.Parse(g.URL)
	if err != nil {
		return Gif{}, fmt.Errorf("not an address: %w", err)
	}
	if address.Scheme != "https" || address.User != nil || !gifHosts.Contains(address.Host) {
		return Gif{}, fmt.Errorf("not a KLIPY address: %q", g.URL)
	}
	if g.Width == 0 || g.Height == 0 || g.Width > maxGifSide || g.Height > maxGifSide {
		return Gif{}, fmt.Errorf("a size of %dx%d", g.Width, g.Height)
	}
	return g, nil
}
