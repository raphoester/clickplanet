package progressionv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1/progressionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/admin_get_progress_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_badges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_fronts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_progress_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/reconcile_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/wear_title_handler"
)

type ProgressionService struct {
	get_titles_handler.GetTitlesHandler
	wear_title_handler.WearTitleHandler
	get_fronts_handler.GetFrontsHandler
	get_badges_handler.GetBadgesHandler
	get_progress_handler.GetProgressHandler
	listen_for_events_handler.ListenForEventsHandler
}

var _ progressionv1connect.ProgressionServiceHandler = ProgressionService{}

type AdminService struct {
	reconcile_titles_handler.ReconcileTitlesHandler
	admin_get_progress_handler.AdminGetProgressHandler
}

var _ progressionv1connect.AdminServiceHandler = AdminService{}
