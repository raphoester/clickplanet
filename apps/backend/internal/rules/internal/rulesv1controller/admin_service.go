package rulesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/rules/v1/rulesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/rulesv1controller/add_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/rulesv1controller/delete_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/rulesv1controller/edit_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/rulesv1controller/get_schedule_handler"
)

type AdminService struct {
	get_schedule_handler.GetScheduleHandler
	add_entry_handler.AddEntryHandler
	edit_entry_handler.EditEntryHandler
	delete_entry_handler.DeleteEntryHandler
}

var _ rulesv1connect.AdminServiceHandler = AdminService{}
