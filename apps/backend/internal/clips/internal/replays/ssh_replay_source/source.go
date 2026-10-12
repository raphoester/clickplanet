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
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

const OpenSSH = "ssh"

// The body goes between single quotes: it holds two times and nothing a shell reads.
const operatorTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
	" --header 'Content-Type: application/json' --post-data '%s'" +
	" http://127.0.0.1:8081/planet.v1.AdminService/GetReplay"

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

	var answered, complained bytes.Buffer
	// "--" ends the options: a host that starts with a dash is a host, never an option of ssh.
	call := exec.CommandContext(ctx, s.client, "--", s.host, fmt.Sprintf(operatorTool, body)) //nolint:gosec // the operator of the job names the host.
	call.Stdout = &answered
	call.Stderr = &complained
	if err := call.Run(); err != nil {
		return replays.Replay{}, fmt.Errorf("ssh failed: %w: %s", err, strings.TrimSpace(complained.String()))
	}

	var replay planetv1.GetReplayResponse
	if err := decoder.Unmarshal(answered.Bytes(), &replay); err != nil {
		return replays.Replay{}, fmt.Errorf("the backend answered what is not a planet.v1.GetReplayResponse: %w", err)
	}
	if replay.GetSince() == nil || replay.GetUntil() == nil {
		return replays.Replay{}, errors.New("the backend answered a replay with no window")
	}

	return replays.Replay{
		Document: answered.Bytes(),
		Events:   len(replay.GetEvents()),
		Window:   replays.Window{Since: replay.GetSince().AsTime(), Until: replay.GetUntil().AsTime()},
	}, nil
}
