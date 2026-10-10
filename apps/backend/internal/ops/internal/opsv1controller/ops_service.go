package opsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1/opsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler"
)

type OpsService struct {
	query_handler.QueryHandler
}

var _ opsv1connect.OpsServiceHandler = OpsService{}
