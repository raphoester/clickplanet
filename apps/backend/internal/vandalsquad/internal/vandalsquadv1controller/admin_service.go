package vandalsquadv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/vandalsquad/v1/vandalsquadv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller/list_graffiti_handler"
)

type AdminService struct {
	list_graffiti_handler.ListGraffitiHandler
}

var _ vandalsquadv1connect.AdminServiceHandler = AdminService{}
