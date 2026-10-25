package seasonsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_race_handler"
)

type AdminService struct {
	get_race_handler.GetRaceHandler
}

var _ seasonsv1connect.AdminServiceHandler = AdminService{}
