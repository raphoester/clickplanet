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
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

const OpenSSH = "ssh"

// The body goes between single quotes: it holds two times and nothing a shell reads.
const operatorTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
	" --header 'Content-Type: application/json' --post-data '%s'" +
	" http://127.0.0.1:8081/planet.v1.AdminService/GetReplay"

// The turn and the round are public: the backend serves them on the listener every browser reaches. The body holds a
// round id, which RoundIDOf keeps to what no shell reads.
const (
	publicTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
		" --header 'Content-Type: application/json' --post-data '%s' http://127.0.0.1:8080/%s"
	turnProcedure  = "gamemaster.v1.GamemasterService/GetTurn"
	roundProcedure = "scoring.v1.ScoringService/GetRound"
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

var _ replays.TurnSource = (*Source)(nil)

func (s *Source) Turn(ctx context.Context) (replays.Turn, error) {
	answered, err := s.ask(ctx, fmt.Sprintf(publicTool, "{}", turnProcedure))
	if err != nil {
		return replays.Turn{}, err
	}

	var turn gamemasterv1.GetTurnResponse
	if err := decoder.Unmarshal(answered, &turn); err != nil {
		return replays.Turn{}, fmt.Errorf("the backend answered what is not a gamemaster.v1.GetTurnResponse: %w", err)
	}

	told, err := debriefOf(turn.GetDebrief())
	if err != nil {
		return replays.Turn{}, err
	}
	told.Document = answered
	told.FinaleAt = finaleOf(turn.GetTimeline())
	return told, nil
}

var errUnknownMoment = errors.New("the backend debriefed a moment the job does not know")

func debriefOf(debrief *gamemasterv1.Debrief) (replays.Turn, error) {
	var (
		label, round string
		endedAt      *timestamppb.Timestamp
	)
	switch moment := debrief.GetMoment().(type) {
	case nil:
		return replays.Turn{}, nil
	case *gamemasterv1.Debrief_Day:
		day := moment.Day
		label, round, endedAt = fmt.Sprintf("%d.%d", day.GetSeason(), day.GetDay()), day.GetRoundId(), day.GetEndedAt()
	case *gamemasterv1.Debrief_Battle:
		battle := moment.Battle
		label, round, endedAt = fmt.Sprintf("%d.battle.%d", battle.GetSeason(), battle.GetNumber()), battle.GetRoundId(), battle.GetEndedAt()
	case *gamemasterv1.Debrief_Season:
		season := moment.Season
		label, round, endedAt = fmt.Sprintf("%d.finale", season.GetSeason()), season.GetFinaleRoundId(), season.GetEndedAt()
		if finale := season.GetFinaleBattle(); finale != nil {
			label = fmt.Sprintf("%d.battle.%d", finale.GetSeason(), finale.GetNumber())
		}
	default:
		return replays.Turn{}, fmt.Errorf("%w: %T", errUnknownMoment, moment)
	}

	told := replays.Turn{Debriefed: label}
	if endedAt.IsValid() {
		told.EndedAt = endedAt.AsTime()
	}
	if round == "" {
		return told, nil
	}
	id, err := replays.RoundIDOf(round)
	if err != nil {
		return replays.Turn{}, fmt.Errorf("the backend debriefed a round the job cannot ask for: %w", err)
	}
	told.Round = id
	return told, nil
}

func finaleOf(timeline *gamemasterv1.Timeline) time.Time {
	for _, battle := range timeline.GetBattles() {
		if battle.GetEndsSeason() {
			return battle.GetStartsAt().AsTime()
		}
	}
	return time.Time{}
}

var _ replays.RoundSource = (*Source)(nil)

var errOtherRound = errors.New("the backend answered another round")

func (s *Source) Round(ctx context.Context, id replays.RoundID) (replays.Round, error) {
	body, err := json.Marshal(map[string]string{"roundId": id.String()})
	if err != nil {
		return replays.Round{}, fmt.Errorf("failed to encode the request: %w", err)
	}

	answered, err := s.ask(ctx, fmt.Sprintf(publicTool, body, roundProcedure))
	if err != nil {
		return replays.Round{}, err
	}

	var round scoringv1.GetRoundResponse
	if err := decoder.Unmarshal(answered, &round); err != nil {
		return replays.Round{}, fmt.Errorf("the backend answered what is not a scoring.v1.GetRoundResponse: %w", err)
	}
	if got := round.GetRound().GetRoundId(); got != id.String() {
		return replays.Round{}, fmt.Errorf("%w: %q, not %q", errOtherRound, got, id)
	}
	return replays.Round{Document: answered, ID: id}, nil
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
