package gamemasterv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/add_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/add_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/delete_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/delete_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/edit_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/edit_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_schedule_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_seasons_handler"
)

type AdminService struct {
	get_schedule_handler.GetScheduleHandler
	add_entry_handler.AddEntryHandler
	edit_entry_handler.EditEntryHandler
	delete_entry_handler.DeleteEntryHandler
	get_seasons_handler.GetSeasonsHandler
	add_season_handler.AddSeasonHandler
	edit_season_handler.EditSeasonHandler
	delete_season_handler.DeleteSeasonHandler
}

var _ gamemasterv1connect.AdminServiceHandler = AdminService{}
