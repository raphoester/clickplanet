package missionsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1/missionsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_missions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/listen_for_events_handler"
)

type MissionService struct {
	get_missions_handler.GetMissionsHandler
	listen_for_events_handler.ListenForEventsHandler
}

var _ missionsv1connect.MissionServiceHandler = MissionService{}
