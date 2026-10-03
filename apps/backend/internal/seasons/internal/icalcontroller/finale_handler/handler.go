package finale_handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Pattern = "GET /seasons/{number}/finale.ics"

const maxAge = 60

type UseCase interface {
	Execute(ctx context.Context, number calendar.Number) (calendar.Season, bool)
}

func New(useCase UseCase, clock cptime.Clock, playURL string) Handler {
	return Handler{useCase: useCase, clock: clock, playURL: playURL}
}

type Handler struct {
	useCase UseCase
	clock   cptime.Clock
	playURL string
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	number, err := strconv.ParseUint(r.PathValue("number"), 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	season, ok := h.useCase.Execute(r.Context(), calendar.Number(number))
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Inline, not attachment: iOS Safari then offers the event to Calendar instead of saving a file.
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="clickplanet-season-%d-finale.ics"`, season.Number))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	_, _ = io.WriteString(w, finaleCalendar(season, h.playURL, h.clock.Now()))
}

func finaleCalendar(season calendar.Season, playURL string, now time.Time) string {
	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//ClickPlanet//Seasons//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"BEGIN:VEVENT",
		fmt.Sprintf("UID:season-%d-finale@clickplanet.lol", season.Number),
		"DTSTAMP:" + utc(now),
		"DTSTART:" + utc(season.FinaleStartsAt),
		"DTEND:" + utc(season.EndsAt),
		"SUMMARY:" + text(fmt.Sprintf("ClickPlanet Season %d: Final Battle", season.Number)),
		"DESCRIPTION:" + text(playURL),
		"URL:" + playURL,
		"END:VEVENT",
		"END:VCALENDAR",
	}

	return strings.Join(lines, "\r\n") + "\r\n"
}

func utc(at time.Time) string {
	return at.UTC().Format("20060102T150405Z")
}

var textEscapes = strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\n", `\n`)

func text(value string) string {
	return textEscapes.Replace(value)
}
