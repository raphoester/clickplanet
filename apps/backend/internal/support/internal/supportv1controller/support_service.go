package supportv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/decide_report_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_reports_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/report_bug_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/report_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/suggest_feature_handler"
)

type SupportService struct {
	suggest_feature_handler.SuggestFeatureHandler
	report_bug_handler.ReportBugHandler
	report_player_handler.ReportPlayerHandler
}

var _ supportv1connect.SupportServiceHandler = SupportService{}

type AdminService struct {
	list_notes_handler.ListNotesHandler
	list_reports_handler.ListReportsHandler
	decide_report_handler.DecideReportHandler
}

var _ supportv1connect.AdminServiceHandler = AdminService{}
