package missionsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1/missionsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/collect_mission_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_missions_handler"
)

type MissionService struct {
	get_missions_handler.GetMissionsHandler
	collect_mission_handler.CollectMissionHandler
}

var _ missionsv1connect.MissionServiceHandler = MissionService{}
