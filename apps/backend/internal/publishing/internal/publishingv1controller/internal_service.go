package publishingv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_posts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/plan_post_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/schedule_post_handler"
)

// What another module asks: the operator's handlers, the same ones, on the internal listener.
type InternalService struct {
	plan_post_handler.PlanPostHandler
	schedule_post_handler.SchedulePostHandler
	list_posts_handler.ListPostsHandler
}

var (
	_ publishingv1connect.InternalServiceHandler = InternalService{}
	_ publishingv1connect.InternalServiceHandler = Off{}
)
