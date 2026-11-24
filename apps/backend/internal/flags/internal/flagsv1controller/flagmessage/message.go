package flagmessage

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

func New(publicBase string) Messages {
	return Messages{base: strings.TrimSuffix(publicBase, "/")}
}

// Messages writes a flag opened since the game shipped as the wire carries it, its picture at its public URL.
type Messages struct {
	base string
}

func (m Messages) Flag(flag catalog.Snapshot) *flagsv1.Flag {
	var closedAt *time.Time
	if flag.State == catalog.Closed {
		closedAt = &flag.DecidedAt
	}
	return &flagsv1.Flag{
		Code: flag.ID.String(), Name: flag.Name.String(), Closed: flag.State == catalog.Closed,
		Sold: m.Sold(flag.Picture, flag.EndsAt, closedAt, flag.Played),
	}
}

func (m Messages) Sold(picture pictures.Snapshot, endsAt time.Time, closedAt *time.Time, played bool) *flagsv1.Sold {
	sold := &flagsv1.Sold{
		Picture: &flagsv1.Picture{
			Url: m.base + "/" + picture.Key(), Width: picture.Width, Height: picture.Height,
			Stretch: picture.Fit.Stretch, FocusX: float32(picture.Fit.FocusX), FocusY: float32(picture.Fit.FocusY),
		},
		EndsAt: timestamppb.New(endsAt),
		Played: played,
	}
	if closedAt != nil {
		sold.ClosedAt = timestamppb.New(*closedAt)
	}
	return sold
}
