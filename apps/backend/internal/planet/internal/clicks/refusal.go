package clicks

import (
	"errors"
	"fmt"
	"time"
)

var ErrRefused = errors.New("the caller is refused")

func NewRefusal(until time.Time) Refusal { return Refusal{until: until} }

type Refusal struct {
	until time.Time
}

func (r Refusal) Error() string {
	return fmt.Sprintf("%s until %s", ErrRefused, r.until.UTC().Format(time.RFC3339))
}

func (r Refusal) Unwrap() error { return ErrRefused }

func (r Refusal) Until() time.Time { return r.until }
