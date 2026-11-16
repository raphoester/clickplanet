package viruses

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

type IDProvider interface {
	NewID() (clicks.VirusID, error)
}
