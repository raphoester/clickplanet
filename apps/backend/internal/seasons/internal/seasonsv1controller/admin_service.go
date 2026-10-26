package seasonsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/add_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/delete_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/edit_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_schedule_handler"
)

type AdminService struct {
	get_schedule_handler.GetScheduleHandler
	add_entry_handler.AddEntryHandler
	edit_entry_handler.EditEntryHandler
	delete_entry_handler.DeleteEntryHandler
}

var _ seasonsv1connect.AdminServiceHandler = AdminService{}
