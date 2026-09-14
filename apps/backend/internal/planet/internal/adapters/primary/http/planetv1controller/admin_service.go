package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/reassign_country_handler"
)

// AdminService is ClickService's counterpart for the loopback admin listener: handlers in a bag.
type AdminService struct {
	reassign_country_handler.ReassignCountryHandler
}

var _ planetv1connect.AdminServiceHandler = AdminService{}
