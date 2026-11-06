package missionsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1/missionsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_mission_totals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/list_flag_days_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/save_rules_handler"
)

type AdminService struct {
	list_flag_days_handler.ListFlagDaysHandler
	get_mission_totals_handler.GetMissionTotalsHandler
	get_rules_handler.GetRulesHandler
	save_rules_handler.SaveRulesHandler
}

var _ missionsv1connect.AdminServiceHandler = AdminService{}
