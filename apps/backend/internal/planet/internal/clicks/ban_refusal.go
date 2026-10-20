package clicks

import (
	"errors"
	"fmt"
	"time"
)

var ErrBanned = errors.New("the caller is banned")

func NewBanRefusal(until time.Time) BanRefusal { return BanRefusal{until: until} }

type BanRefusal struct {
	until time.Time
}

func (r BanRefusal) Error() string {
	return fmt.Sprintf("%s until %s", ErrBanned, r.until.UTC().Format(time.RFC3339))
}

func (r BanRefusal) Unwrap() error { return ErrBanned }

func (r BanRefusal) Until() time.Time { return r.until }
