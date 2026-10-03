package finale_handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/icalcontroller/finale_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type numberedSeasons map[calendar.Number]calendar.Season

func (s numberedSeasons) Execute(_ context.Context, number calendar.Number) (calendar.Season, bool) {
	season, ok := s[number]
	return season, ok
}

var (
	seasons = numberedSeasons{
		0: {
			Number:         0,
			FinaleStartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
			EndsAt:         time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC),
		},
		2: {
			Number:         2,
			FinaleStartsAt: time.Date(2027, 1, 31, 20, 30, 0, 0, time.UTC),
			EndsAt:         time.Date(2027, 1, 31, 23, 0, 0, 0, time.UTC),
		},
	}
	now = time.Date(2026, 10, 3, 10, 15, 30, 250_000_000, time.UTC)
)

func get(t *testing.T, playURL, path string) *httptest.ResponseRecorder {
	t.Helper()

	router := http.NewServeMux()
	router.Handle(finale_handler.Pattern, finale_handler.New(seasons, cptime.NewFixedClock(now), playURL))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))

	return recorder
}

func TestTheFinaleIsOneEventInUTCWithALinkToTheGame(t *testing.T) {
	res := get(t, "https://clickplanet.lol/play", "/seasons/0/finale.ics")

	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "text/calendar; charset=utf-8", res.Header().Get("Content-Type"))
	assert.Equal(t, `inline; filename="clickplanet-season-0-finale.ics"`, res.Header().Get("Content-Disposition"))
	assert.Equal(t, "public, max-age=60", res.Header().Get("Cache-Control"))
	assert.Equal(t, strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//ClickPlanet//Seasons//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:season-0-finale@clickplanet.lol",
		"DTSTAMP:20261003T101530Z",
		"DTSTART:20261031T210000Z",
		"DTEND:20261031T230000Z",
		"SUMMARY:ClickPlanet Season 0: Final Battle",
		"DESCRIPTION:https://clickplanet.lol/play",
		"URL:https://clickplanet.lol/play",
		"END:VEVENT",
		"END:VCALENDAR",
		"",
	}, "\r\n"), res.Body.String())
}

func TestTheFileTakesItsDatesAndItsNumberFromTheSeason(t *testing.T) {
	res := get(t, "https://clickplanet.lol/play", "/seasons/2/finale.ics")

	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `inline; filename="clickplanet-season-2-finale.ics"`, res.Header().Get("Content-Disposition"))
	assert.Contains(t, res.Body.String(), "\r\nUID:season-2-finale@clickplanet.lol\r\n")
	assert.Contains(t, res.Body.String(), "\r\nDTSTART:20270131T203000Z\r\nDTEND:20270131T230000Z\r\n")
	assert.Contains(t, res.Body.String(), "\r\nSUMMARY:ClickPlanet Season 2: Final Battle\r\n")
}

func TestATextValueIsEscaped(t *testing.T) {
	res := get(t, `https://example.com/play?a=1,b;c\d`, "/seasons/0/finale.ics")

	assert.Contains(t, res.Body.String(), "\r\nDESCRIPTION:https://example.com/play?a=1\\,b\\;c\\\\d\r\n")
}

func TestEveryLineIsWithin75Octets(t *testing.T) {
	res := get(t, "https://clickplanet.lol/play", "/seasons/0/finale.ics")

	for _, line := range strings.Split(res.Body.String(), "\r\n") {
		assert.LessOrEqual(t, len(line), 75, line)
	}
}

func TestASeasonNotInTheCalendarIsNotFound(t *testing.T) {
	for _, path := range []string{"/seasons/1/finale.ics", "/seasons/zero/finale.ics", "/seasons/-1/finale.ics", "/seasons/4294967296/finale.ics"} {
		assert.Equal(t, http.StatusNotFound, get(t, "https://clickplanet.lol/play", path).Code, path)
	}
}
