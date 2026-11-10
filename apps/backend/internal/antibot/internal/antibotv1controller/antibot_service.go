package antibotv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/acknowledge_warning_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/listen_for_events_handler"
)

type AntibotService struct {
	listen_for_events_handler.ListenForEventsHandler
	acknowledge_warning_handler.AcknowledgeWarningHandler
}

var _ antibotv1connect.AntibotServiceHandler = AntibotService{}
