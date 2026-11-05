package ssh_appearance_store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/keep_appearances_usecase"
)

const OpenSSH = "ssh"

// The body comes on the standard input, never the command line: it holds the model's headline, which no shell should
// read.
const operatorTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
	" --header 'Content-Type: application/json' --post-file=/dev/stdin" +
	" http://127.0.0.1:8081/clips.v1.AdminService/KeepAppearance"

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

// host: user@host of the machine the backend runs on. client: the ssh program.
func New(host, client string) (*Store, error) {
	if host == "" {
		return nil, errors.New("the ssh host is empty")
	}
	return &Store{host: host, client: client}, nil
}

type Store struct {
	host   string
	client string
}

var _ keep_appearances_usecase.Appearances = (*Store)(nil)

func (s *Store) Keep(ctx context.Context, appearance posts.Appearance) (int, error) {
	request := &clipsv1.KeepAppearanceRequest{
		CastId: appearance.Cast, PostId: string(appearance.Post), Title: appearance.Title,
		ScheduledAt: timestamppb.New(appearance.At),
	}
	for _, number := range appearance.Fighters {
		request.Fighters = append(request.Fighters, uint32(number)) //nolint:gosec // a number of the cast, from 1.
	}
	for _, platform := range appearance.Platforms {
		request.Platforms = append(request.Platforms, string(platform))
	}
	body, err := protojson.Marshal(request)
	if err != nil {
		return 0, fmt.Errorf("failed to write the request: %w", err)
	}

	answered, err := s.ask(ctx, body)
	if err != nil {
		return 0, err
	}
	var kept clipsv1.KeepAppearanceResponse
	if err := decoder.Unmarshal(answered, &kept); err != nil {
		return 0, fmt.Errorf("the backend answered what is not a clips.v1.KeepAppearanceResponse: %w", err)
	}
	return int(kept.GetKept()), nil
}

func (s *Store) ask(ctx context.Context, body []byte) ([]byte, error) {
	var answered, complained bytes.Buffer
	// "--" ends the options: a host that starts with a dash is a host, never an option of ssh.
	call := exec.CommandContext(ctx, s.client, "--", s.host, operatorTool) //nolint:gosec // the operator of the job names the host.
	call.Stdin = bytes.NewReader(body)
	call.Stdout = &answered
	call.Stderr = &complained
	if err := call.Run(); err != nil {
		return nil, fmt.Errorf("ssh failed: %w: %s", err, strings.TrimSpace(complained.String()))
	}
	return answered.Bytes(), nil
}
