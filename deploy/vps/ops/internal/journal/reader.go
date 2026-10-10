package journal

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

var ErrForeignTag = errors.New("the tag is not one of this stack's containers")

var tagShape = regexp.MustCompile(`^cp-[a-z0-9-]+$`)

type Tag string

func ParseTag(raw string) (Tag, error) {
	if !tagShape.MatchString(raw) {
		return "", ErrForeignTag
	}
	return Tag(raw), nil
}

type Query struct {
	Tag      Tag
	Since    time.Time
	Until    time.Time
	Contains string
	Limit    excerpt.Limit
}

type Reader interface {
	Read(ctx context.Context, query Query) (excerpt.Excerpt, error)
}
