package random_flag_ids

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

type Provider struct{}

var _ catalog.IDProvider = Provider{}

func (Provider) NewID() (catalog.FlagID, error) {
	body := make([]byte, catalog.FlagIDLength)
	size := big.NewInt(int64(len(alphabet)))
	for i := range body {
		drawn, err := rand.Int(rand.Reader, size)
		if err != nil {
			return "", fmt.Errorf("failed to draw a flag id: %w", err)
		}
		body[i] = alphabet[drawn.Int64()]
	}
	return catalog.ParseFlagID(catalog.FlagIDPrefix + string(body)) //nolint:wrapcheck // the parse names it.
}
