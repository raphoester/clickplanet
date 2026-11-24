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
		Sold: m.Sold(flag.Picture, flag.Pin, flag.EndsAt, closedAt),
	}
}

func (m Messages) Sold(picture pictures.Snapshot, pin catalog.Pin, endsAt time.Time, closedAt *time.Time) *flagsv1.Sold {
	sold := &flagsv1.Sold{
		Picture: m.Picture(picture),
		EndsAt:  timestamppb.New(endsAt),
		Pin:     Pin(pin),
	}
	if closedAt != nil {
		sold.ClosedAt = timestamppb.New(*closedAt)
	}
	return sold
}

// Picture is a picture made public, at its address.
func (m Messages) Picture(picture pictures.Snapshot) *flagsv1.Picture {
	return PictureAt(m.base+"/"+picture.Key(), picture)
}

func PictureAt(url string, picture pictures.Snapshot) *flagsv1.Picture {
	return &flagsv1.Picture{
		Url: url, Width: picture.Width, Height: picture.Height,
		Stretch: picture.Fit.Stretch, FocusX: float32(picture.Fit.FocusX), FocusY: float32(picture.Fit.FocusY),
	}
}

func Pin(pin catalog.Pin) *flagsv1.Pin {
	return &flagsv1.Pin{Latitude: pin.Latitude, Longitude: pin.Longitude}
}
