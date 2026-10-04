package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_feed_start_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/read_log_handler"
)

type InternalService struct {
	get_feed_start_handler.GetFeedStartHandler
	read_log_handler.ReadLogHandler
}

var _ planetv1connect.InternalServiceHandler = InternalService{}
