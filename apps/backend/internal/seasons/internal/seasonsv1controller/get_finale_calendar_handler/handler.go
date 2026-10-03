package get_finale_calendar_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ics "github.com/arran4/golang-ical"
	"google.golang.org/genproto/googleapis/api/httpbody"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type UseCase interface {
	Execute(ctx context.Context, number calendar.Number) (calendar.Season, bool)
}

func New(useCase UseCase, clock cptime.Clock, playURL string) GetFinaleCalendarHandler {
	return GetFinaleCalendarHandler{useCase: useCase, clock: clock, playURL: playURL}
}

type GetFinaleCalendarHandler struct {
	useCase UseCase
	clock   cptime.Clock
	playURL string
}

func (h GetFinaleCalendarHandler) GetFinaleCalendar(
	ctx context.Context,
	req *connect.Request[seasonsv1.GetFinaleCalendarRequest],
) (*connect.Response[httpbody.HttpBody], error) {
	season, ok := h.useCase.Execute(ctx, calendar.Number(req.Msg.GetNumber()))
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("there is no season %d", req.Msg.GetNumber()))
	}

	res := connect.NewResponse(&httpbody.HttpBody{
		ContentType: "text/calendar; charset=utf-8",
		// RFC 5545 ends every line with CRLF; the library's default follows the OS.
		Data: []byte(h.finale(season).Serialize(ics.WithNewLineWindows)),
	})
	// Inline, not attachment: iOS Safari then offers the event to Calendar instead of saving a file.
	res.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="clickplanet-season-%d-finale.ics"`, season.Number))

	return res, nil
}

func (h GetFinaleCalendarHandler) finale(season calendar.Season) *ics.Calendar {
	finale := ics.NewCalendar()
	finale.SetProductId("-//ClickPlanet//Seasons//EN")
	finale.SetCalscale("GREGORIAN")
	finale.SetMethod(ics.MethodPublish)

	event := finale.AddEvent(fmt.Sprintf("season-%d-finale@clickplanet.lol", season.Number))
	event.SetDtStampTime(h.clock.Now())
	event.SetStartAt(season.FinaleStartsAt)
	event.SetEndAt(season.EndsAt)
	event.SetSummary(fmt.Sprintf("ClickPlanet Season %d: Final Battle", season.Number))
	event.SetDescription(h.playURL)
	event.SetURL(h.playURL)

	return finale
}
