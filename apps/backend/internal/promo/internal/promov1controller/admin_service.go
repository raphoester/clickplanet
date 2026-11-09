package promov1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1/promov1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/create_background_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_cards_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/keep_background_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_backgrounds_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_due_teasers_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_teasers_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/post_teaser_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/remove_background_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/save_cards_handler"
)

type AdminService struct {
	create_background_upload_handler.CreateBackgroundUploadHandler
	keep_background_handler.KeepBackgroundHandler
	list_backgrounds_handler.ListBackgroundsHandler
	remove_background_handler.RemoveBackgroundHandler
	get_cards_handler.GetCardsHandler
	save_cards_handler.SaveCardsHandler
	list_teasers_handler.ListTeasersHandler
	list_due_teasers_handler.ListDueTeasersHandler
	post_teaser_handler.PostTeaserHandler
}

var _ promov1connect.AdminServiceHandler = AdminService{}
