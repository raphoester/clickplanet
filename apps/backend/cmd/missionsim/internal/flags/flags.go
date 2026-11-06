package flags

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions"
)

const usage = `usage:
  missionsim fetch -since <2026-11-01T06:00:00Z> -out <folder> [-ssh <user@host>]
  missionsim run -data <folder> [-sizings <sizings.json>] [-rows <rows.csv>]

fetch copies the days told and every act of the ledger from -since, a season's start when its map was blank, out of
production's postgres over ssh, into the folder: CLICKPLANET_SSH stands in for -ssh. It only reads.
run replays that season and draws each flag's mission every day as the game does, with the sizing the game plays and
each one of -sizings, and prints by flag size and kind how many were done by the players playing as they did, how
fast, how big the goal was beside the flag's day, and the points; -rows writes every flag's day. A sizings file is a
JSON array: [{"name": "harder", "goal_percent": 80, "least_conquest": 300, "least_shields": 300,
"fewest_points": 2, "most_points": 6}].`

type Command string

const (
	Fetch Command = "fetch"
	Run   Command = "run"
)

const HostVariable = "CLICKPLANET_SSH"

var ErrArguments = errors.New("wrong arguments")

type Config struct {
	Command Command
	Host    string
	Since   time.Time
	Folder  string
	Sizings []missions.NamedSizing
	Rows    string
}

func Parse(args []string) (Config, error) {
	if len(args) == 0 {
		return Config{}, fmt.Errorf("%w\n%s", ErrArguments, usage)
	}
	switch Command(args[0]) {
	case Fetch:
		return parseFetch(args[1:])
	case Run:
		return parseRun(args[1:])
	default:
		return Config{}, fmt.Errorf("%w: no command %q\n%s", ErrArguments, args[0], usage)
	}
}

func parseFetch(args []string) (Config, error) {
	set := flag.NewFlagSet(string(Fetch), flag.ContinueOnError)
	host := set.String("ssh", os.Getenv(HostVariable), "user@host of production")
	since := set.String("since", "", "the season's start, RFC 3339")
	folder := set.String("out", "", "the folder to write the season in")
	if err := set.Parse(args); err != nil {
		return Config{}, fmt.Errorf("%w: %w\n%s", ErrArguments, err, usage)
	}
	if *host == "" || *folder == "" {
		return Config{}, fmt.Errorf("%w: fetch needs -ssh (or %s) and -out\n%s", ErrArguments, HostVariable, usage)
	}
	start, err := time.Parse(time.RFC3339, *since)
	if err != nil {
		return Config{}, fmt.Errorf("%w: -since: %w\n%s", ErrArguments, err, usage)
	}
	return Config{Command: Fetch, Host: *host, Since: start, Folder: *folder}, nil
}

func parseRun(args []string) (Config, error) {
	set := flag.NewFlagSet(string(Run), flag.ContinueOnError)
	folder := set.String("data", "", "the folder fetch wrote")
	sizings := set.String("sizings", "", "a JSON file of the sizings to try")
	rows := set.String("rows", "", "a CSV file to write every flag's day in")
	if err := set.Parse(args); err != nil {
		return Config{}, fmt.Errorf("%w: %w\n%s", ErrArguments, err, usage)
	}
	if *folder == "" {
		return Config{}, fmt.Errorf("%w: run needs -data\n%s", ErrArguments, usage)
	}
	config := Config{Command: Run, Folder: *folder, Rows: *rows}
	if *sizings != "" {
		read, err := sizingsOf(*sizings)
		if err != nil {
			return Config{}, err
		}
		config.Sizings = read
	}
	return config, nil
}

type sizingRecord struct {
	Name          string `json:"name"`
	GoalPercent   uint64 `json:"goal_percent"`
	LeastConquest uint64 `json:"least_conquest"`
	LeastShields  uint64 `json:"least_shields"`
	FewestPoints  uint32 `json:"fewest_points"`
	MostPoints    uint32 `json:"most_points"`
}

func sizingsOf(path string) ([]missions.NamedSizing, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // the operator names the file.
	if err != nil {
		return nil, fmt.Errorf("failed to read the sizings: %w", err)
	}
	var records []sizingRecord
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("%w: %s is not a JSON array of sizings: %w", ErrArguments, path, err)
	}
	named := make([]missions.NamedSizing, 0, len(records))
	for _, record := range records {
		if record.Name == "" || record.Name == missions.Played {
			return nil, fmt.Errorf("%w: a sizing needs a name other than %q", ErrArguments, missions.Played)
		}
		named = append(named, missions.NamedSizing{Name: record.Name, Sizing: missions.Sizing{
			GoalPercent: record.GoalPercent, LeastConquest: record.LeastConquest, LeastShields: record.LeastShields,
			FewestPoints: record.FewestPoints, MostPoints: record.MostPoints,
		}})
	}
	return named, nil
}
