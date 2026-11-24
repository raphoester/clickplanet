package keep_played_flag_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
)

type Catalog interface {
	Unkept(ctx context.Context, flag string) (bool, error)
	Keep(ctx context.Context, flag string) error
}

type Commands interface {
	Send(ctx context.Context, commands ...proto.Message) error
}

type In struct {
	Flag     string
	PlayedAt time.Time
}

func New(catalog Catalog, commands Commands) *UseCase {
	return &UseCase{catalog: catalog, commands: commands}
}

type UseCase struct {
	catalog  Catalog
	commands Commands
}

// Execute tells the flags module, once, that a player played for a flag opened since the game shipped: from then on it
// is never deleted. Sent before it is noted, so a crash between the two sends it again, which changes nothing.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	unkept, err := u.catalog.Unkept(ctx, in.Flag)
	if err != nil {
		return fmt.Errorf("failed to read whether %q was kept: %w", in.Flag, err)
	}
	if !unkept {
		return nil
	}
	if err := u.commands.Send(ctx, &flagsv1.KeepFlag{Flag: in.Flag, PlayedAt: timestamppb.New(in.PlayedAt)}); err != nil {
		return fmt.Errorf("failed to ask the flags module to keep %q: %w", in.Flag, err)
	}
	if err := u.catalog.Keep(ctx, in.Flag); err != nil {
		return fmt.Errorf("failed to note that %q was kept: %w", in.Flag, err)
	}
	return nil
}
