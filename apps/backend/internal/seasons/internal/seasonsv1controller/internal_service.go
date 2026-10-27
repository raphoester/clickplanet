package seasonsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_rule_set_names_handler"
)

type InternalService struct {
	get_rule_set_names_handler.GetRuleSetNamesHandler
}

var _ seasonsv1connect.InternalServiceHandler = InternalService{}
