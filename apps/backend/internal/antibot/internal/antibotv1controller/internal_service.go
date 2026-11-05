package antibotv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rule_set_names_handler"
)

type InternalService struct {
	get_rule_set_names_handler.GetRuleSetNamesHandler
}

var _ antibotv1connect.InternalServiceHandler = InternalService{}
