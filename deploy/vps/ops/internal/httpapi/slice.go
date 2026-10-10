package httpapi

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

const (
	usualLimit   = 1000
	highestLimit = 20000
	mostBytes    = 8 << 20
)

var (
	errNoSince       = errors.New("since is required, as an RFC 3339 time")
	errBackwards     = errors.New("until is before since")
	errLimitOutOfRun = fmt.Errorf("limit must be between 1 and %d", highestLimit)
)

type slice struct {
	since    time.Time
	until    time.Time
	contains string
	limit    excerpt.Limit
}

func sliceOf(values url.Values, now time.Time) (slice, error) {
	if values.Get("since") == "" {
		return slice{}, errNoSince
	}
	since, err := time.Parse(time.RFC3339, values.Get("since"))
	if err != nil {
		return slice{}, fmt.Errorf("since is not an RFC 3339 time: %w", err)
	}

	until := now
	if raw := values.Get("until"); raw != "" {
		until, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return slice{}, fmt.Errorf("until is not an RFC 3339 time: %w", err)
		}
	}
	if until.Before(since) {
		return slice{}, errBackwards
	}

	limit, err := limitOf(values)
	if err != nil {
		return slice{}, err
	}
	return slice{since: since, until: until, contains: values.Get("contains"), limit: limit}, nil
}

func limitOf(values url.Values) (excerpt.Limit, error) {
	lines := usualLimit
	if raw := values.Get("limit"); raw != "" {
		asked, err := strconv.Atoi(raw)
		if err != nil || asked < 1 || asked > highestLimit {
			return excerpt.Limit{}, errLimitOutOfRun
		}
		lines = asked
	}
	return excerpt.Limit{Lines: lines, Bytes: mostBytes}, nil
}
