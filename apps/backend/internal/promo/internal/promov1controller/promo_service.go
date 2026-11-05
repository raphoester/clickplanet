package promov1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1/promov1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler"
)

type PromoService struct {
	get_teaser_handler.GetTeaserHandler
}

var _ promov1connect.PromoServiceHandler = PromoService{}
