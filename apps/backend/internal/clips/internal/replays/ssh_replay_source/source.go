package ssh_replay_source

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

const OpenSSH = "ssh"

// The body goes between single quotes: it holds two times and nothing a shell reads.
const operatorTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
	" --header 'Content-Type: application/json' --post-data '%s'" +
	" http://127.0.0.1:8081/planet.v1.AdminService/GetReplay"

// The race and the season are public: the backend serves them on the listener every browser reaches.
const (
	publicTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
		" --header 'Content-Type: application/json' --post-data '{}' http://127.0.0.1:8080/"
	raceTool   = publicTool + "seasons.v1.SeasonService/GetRace"
	seasonTool = publicTool + "seasons.v1.SeasonService/GetSeason"
)

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

// host: user@host of the machine the backend runs on. client: the ssh program.
func New(host, client string) (*Source, error) {
	if host == "" {
		return nil, errors.New("the ssh host is empty")
	}
	return &Source{host: host, client: client}, nil
}

type Source struct {
	host   string
	client string
}

var _ replays.Source = (*Source)(nil)

func (s *Source) Replay(ctx context.Context, window replays.Window) (replays.Replay, error) {
	body, err := json.Marshal(map[string]string{
		"since": window.Since.UTC().Format(time.RFC3339Nano),
		"until": window.Until.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return replays.Replay{}, fmt.Errorf("failed to encode the request: %w", err)
	}

	answered, err := s.ask(ctx, fmt.Sprintf(operatorTool, body))
	if err != nil {
		return replays.Replay{}, err
	}

	var replay planetv1.GetReplayResponse
	if err := decoder.Unmarshal(answered, &replay); err != nil {
		return replays.Replay{}, fmt.Errorf("the backend answered what is not a planet.v1.GetReplayResponse: %w", err)
	}
	if replay.GetSince() == nil || replay.GetUntil() == nil {
		return replays.Replay{}, errors.New("the backend answered a replay with no window")
	}

	return replays.Replay{
		Document: answered,
		Events:   len(replay.GetEvents()),
		Window:   replays.Window{Since: replay.GetSince().AsTime(), Until: replay.GetUntil().AsTime()},
	}, nil
}

var _ replays.RaceSource = (*Source)(nil)

func (s *Source) Race(ctx context.Context) (replays.Race, error) {
	answered, err := s.ask(ctx, raceTool)
	if err != nil {
		return replays.Race{}, err
	}

	var race seasonsv1.GetRaceResponse
	if err := decoder.Unmarshal(answered, &race); err != nil {
		return replays.Race{}, fmt.Errorf("the backend answered what is not a seasons.v1.GetRaceResponse: %w", err)
	}

	closed := race.GetRace().GetClosed()
	if closed == nil {
		return replays.Race{Document: answered}, nil
	}
	return replays.Race{
		Document: answered,
		Closed:   fmt.Sprintf("%d.%d", closed.GetSeason(), closed.GetNumber()),
		EndedAt:  time.UnixMilli(closed.GetEndedAtUnixMs()).UTC(),
	}, nil
}

var _ replays.SeasonSource = (*Source)(nil)

func (s *Source) Season(ctx context.Context) (replays.Season, error) {
	answered, err := s.ask(ctx, seasonTool)
	if err != nil {
		return replays.Season{}, err
	}

	var season seasonsv1.GetSeasonResponse
	if err := decoder.Unmarshal(answered, &season); err != nil {
		return replays.Season{}, fmt.Errorf("the backend answered what is not a seasons.v1.GetSeasonResponse: %w", err)
	}

	current := season.GetSeason()
	if current == nil {
		return replays.Season{Document: answered}, nil
	}
	return replays.Season{Document: answered, FinaleStartsAt: time.UnixMilli(current.GetFinaleStartsAtUnixMs()).UTC()}, nil
}

func (s *Source) ask(ctx context.Context, command string) ([]byte, error) {
	var answered, complained bytes.Buffer
	// "--" ends the options: a host that starts with a dash is a host, never an option of ssh.
	call := exec.CommandContext(ctx, s.client, "--", s.host, command) //nolint:gosec // the operator of the job names the host.
	call.Stdout = &answered
	call.Stderr = &complained
	if err := call.Run(); err != nil {
		return nil, fmt.Errorf("ssh failed: %w: %s", err, strings.TrimSpace(complained.String()))
	}
	return answered.Bytes(), nil
}
