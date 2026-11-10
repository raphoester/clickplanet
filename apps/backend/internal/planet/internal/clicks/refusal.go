package clicks

import (
	"errors"
	"fmt"
	"time"
)

var ErrRefused = errors.New("the caller is refused")

func NewRefusal(until time.Time) Refusal { return Refusal{until: until} }

func NewWarning() Refusal { return Refusal{warning: true} }

type Refusal struct {
	until   time.Time
	warning bool
}

func (r Refusal) Error() string {
	if r.warning {
		return fmt.Sprintf("%s until it acknowledges its warning", ErrRefused)
	}
	return fmt.Sprintf("%s until %s", ErrRefused, r.until.UTC().Format(time.RFC3339))
}

func (r Refusal) Unwrap() error { return ErrRefused }

func (r Refusal) Until() time.Time { return r.until }

func (r Refusal) Warning() bool { return r.warning }
