package metagamev1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1/metagamev1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/add_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/delete_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/edit_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_schedule_handler"
)

type AdminService struct {
	get_schedule_handler.GetScheduleHandler
	add_entry_handler.AddEntryHandler
	edit_entry_handler.EditEntryHandler
	delete_entry_handler.DeleteEntryHandler
}

var _ metagamev1connect.AdminServiceHandler = AdminService{}
