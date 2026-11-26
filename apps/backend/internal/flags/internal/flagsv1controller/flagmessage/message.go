package flagmessage

import (
	"strings"

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

// Flag is the flag as every page reads it: never its holder or its removal's note.
func (m Messages) Flag(flag catalog.Snapshot) *flagsv1.Flag {
	sold := &flagsv1.Sold{
		Picture: m.Picture(flag.Picture),
		EndsAt:  timestamppb.New(flag.EndsAt),
		Pin:     Pin(flag.Pin),
		Removed: flag.State == catalog.Removed,
	}
	if flag.State != catalog.Open {
		sold.ClosedAt = timestamppb.New(flag.DecidedAt)
	}
	return &flagsv1.Flag{
		Code: flag.ID.String(), Name: flag.Name.String(), Closed: flag.State != catalog.Open, Sold: sold,
	}
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
