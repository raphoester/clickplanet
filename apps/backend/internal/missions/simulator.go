package missions

import (
	"context"
	"fmt"
	"time"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation/csv_season"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation/ssh_season_dump"
)

// The simulator's words, for cmd/missionsim.
type (
	Sizing           = flagmissions.Sizing
	NamedSizing      = simulation.Named
	SimulationReport = simulation.Report
	SizingReport     = simulation.SizingReport
	SimulationRow    = simulation.Row
	SimulationLine   = simulation.Line
)

// The name the report gives the sizing the game plays.
const Played = "played"

// Copies the days told and the ledger's acts from since, a season's start, out of production's postgres over ssh,
// into the folder Simulate reads.
func FetchSeason(ctx context.Context, host string, since time.Time, folder string) error {
	dump, err := ssh_season_dump.New(host, ssh_season_dump.OpenSSH)
	if err != nil {
		return err //nolint:wrapcheck // it says what is wrong with the host.
	}
	return dump.Write(ctx, since, folder) //nolint:wrapcheck // it names what it could not copy.
}

// Replays the season FetchSeason copied with the sizing the game plays, then with each other one.
func Simulate(folder string, others []NamedSizing) (SimulationReport, error) {
	sizings := []NamedSizing{{Name: Played, Sizing: flagmissions.Played}}
	for _, other := range others {
		if err := other.Sizing.Validate(); err != nil {
			return SimulationReport{}, fmt.Errorf("sizing %q: %w", other.Name, err)
		}
		sizings = append(sizings, other)
	}

	globe, err := shippedGlobe()
	if err != nil {
		return SimulationReport{}, err
	}
	calendar, acts, err := csv_season.Read(folder)
	if err != nil {
		return SimulationReport{}, err //nolint:wrapcheck // it names the file and the line.
	}
	return simulation.Run(calendar, acts, globe, sizings) //nolint:wrapcheck // it names the act.
}

// A blank map on the land of the map this build ships, the one the ledger's tiles number since the last remap.
func shippedGlobe() (*simulation.Globe, error) {
	blob, asset, err := mapdata.Borders()
	if err != nil {
		return nil, fmt.Errorf("failed to read the borders: %w", err)
	}
	regions, codes, err := mapdata.DecodeBorders(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", asset, err)
	}
	grounds := make([]string, len(regions))
	for n, region := range regions {
		if int(region) >= len(codes) {
			return nil, fmt.Errorf("%s puts tile %d in region %d, past the %d it names", asset, n+1, region, len(codes))
		}
		grounds[n] = codes[region]
	}
	return simulation.NewGlobe(grounds) //nolint:wrapcheck // it names the tile.
}
