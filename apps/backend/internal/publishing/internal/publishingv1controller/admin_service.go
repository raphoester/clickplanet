package publishingv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/create_media_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_room_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_posts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/plan_post_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/schedule_post_handler"
)

type AdminService struct {
	create_media_upload_handler.CreateMediaUploadHandler
	get_room_handler.GetRoomHandler
	plan_post_handler.PlanPostHandler
	schedule_post_handler.SchedulePostHandler
	list_posts_handler.ListPostsHandler
}

var _ publishingv1connect.AdminServiceHandler = AdminService{}
