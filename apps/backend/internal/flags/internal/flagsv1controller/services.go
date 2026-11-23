package flagsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler"
)

type FlagService struct {
	list_flags_handler.ListFlagsHandler
}

var _ flagsv1connect.FlagServiceHandler = FlagService{}

type AdminService struct {
	list_flags_handler.ListFlagsHandler
}

var _ flagsv1connect.AdminServiceHandler = AdminService{}
