package boxesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1/boxesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/claim_box_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/listen_for_events_handler"
)

type BoxService struct {
	claim_box_handler.ClaimBoxHandler
	listen_for_events_handler.ListenForEventsHandler
}

var _ boxesv1connect.BoxServiceHandler = BoxService{}
