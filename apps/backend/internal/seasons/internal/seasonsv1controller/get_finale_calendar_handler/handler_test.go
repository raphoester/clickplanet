package get_finale_calendar_handler_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/httpbody"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_finale_calendar_handler"
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

func finale(t *testing.T, playURL string, number uint32) (*connect.Response[httpbody.HttpBody], error) {
	t.Helper()

	return get_finale_calendar_handler.New(seasons, cptime.NewFixedClock(now), playURL). //nolint:wrapcheck // the test reads the connect code.
												GetFinaleCalendar(t.Context(), connect.NewRequest(&seasonsv1.GetFinaleCalendarRequest{Number: number}))
}

func TestTheFinaleIsOneEventInUTCWithALinkToTheGame(t *testing.T) {
	res, err := finale(t, "https://clickplanet.lol/play", 0)
	require.NoError(t, err)

	assert.Equal(t, "text/calendar; charset=utf-8", res.Msg.GetContentType())
	assert.Equal(t, `inline; filename="clickplanet-season-0-finale.ics"`, res.Header().Get("Content-Disposition"))
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
	}, "\r\n"), string(res.Msg.GetData()))
}

func TestTheFileTakesItsDatesAndItsNumberFromTheSeason(t *testing.T) {
	res, err := finale(t, "https://clickplanet.lol/play", 2)
	require.NoError(t, err)

	data := string(res.Msg.GetData())
	assert.Equal(t, `inline; filename="clickplanet-season-2-finale.ics"`, res.Header().Get("Content-Disposition"))
	assert.Contains(t, data, "\r\nUID:season-2-finale@clickplanet.lol\r\n")
	assert.Contains(t, data, "\r\nDTSTART:20270131T203000Z\r\nDTEND:20270131T230000Z\r\n")
	assert.Contains(t, data, "\r\nSUMMARY:ClickPlanet Season 2: Final Battle\r\n")
}

func TestATextValueIsEscaped(t *testing.T) {
	res, err := finale(t, `https://example.com/play?a=1,b;c\d`, 0)
	require.NoError(t, err)

	assert.Contains(t, string(res.Msg.GetData()), "\r\nDESCRIPTION:https://example.com/play?a=1\\,b\\;c\\\\d\r\n")
}

func TestALongLineIsFoldedWithin75Octets(t *testing.T) {
	res, err := finale(t, "https://clickplanet.lol/play?"+strings.Repeat("x", 100), 0)
	require.NoError(t, err)

	for _, line := range strings.Split(string(res.Msg.GetData()), "\r\n") {
		assert.LessOrEqual(t, len(line), 75, line)
	}
}

func TestASeasonNotInTheCalendarIsNotFound(t *testing.T) {
	_, err := finale(t, "https://clickplanet.lol/play", 1)

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
