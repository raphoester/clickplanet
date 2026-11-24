package open_flag_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Gallery interface {
	Draw(ctx context.Context, id pictures.UploadID) (pictures.Picture, error)
	Discard(ctx context.Context, id pictures.UploadID) error
}

type Flags interface {
	Save(ctx context.Context, flags ...catalog.Flag) error
}

type In struct {
	Name      string
	Upload    string
	Latitude  float64
	Longitude float64
	EndsAt    time.Time
}

const timeout = 30 * time.Second

func New(
	opening catalog.Opening, gallery Gallery, db cpatomic.Transactioner, flags Flags, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{opening: opening, gallery: gallery, db: db, flags: flags, outbox: outbox}
}

type UseCase struct {
	opening catalog.Opening
	gallery Gallery
	db      cpatomic.Transactioner
	flags   Flags
	outbox  cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context, in In) (catalog.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name, err := catalog.ParseName(in.Name)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	pin, err := catalog.ParsePin(in.Latitude, in.Longitude)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	upload, err := pictures.ParseUploadID(in.Upload)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	picture, err := u.gallery.Draw(ctx, upload)
	if err != nil {
		return catalog.Snapshot{}, fmt.Errorf("failed to draw the flag's picture: %w", err)
	}
	flag, err := u.opening.Open(name, picture, pin, in.EndsAt)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the opening names it.
	}

	// Never a flag the other modules are not told of, nor a fact of a flag not kept.
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		if err := u.flags.Save(ctx, flag); err != nil {
			return fmt.Errorf("failed to keep the flag: %w", err)
		}
		if err := u.outbox.Record(ctx, flag.Facts()...); err != nil {
			return fmt.Errorf("failed to record that the flag was opened: %w", err)
		}
		return nil
	}); err != nil {
		return catalog.Snapshot{}, fmt.Errorf("failed to open the flag %q: %w", name, err)
	}
	// The flag is open: a file left in the private folder harms nobody.
	_ = u.gallery.Discard(ctx, upload)
	return flag.TakeSnapshot(), nil
}
