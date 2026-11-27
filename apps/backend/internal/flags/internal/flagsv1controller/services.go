package flagsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/add_time_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/appeal_submission_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/approve_submission_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/close_flag_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/create_flag_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/create_submission_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/get_payable_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_my_submissions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_submissions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/open_flag_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/open_submitted_flag_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/reject_submission_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/rename_flag_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submit_flag_handler"
)

type FlagService struct {
	list_flags_handler.ListFlagsHandler
	listen_for_events_handler.ListenForEventsHandler
	create_submission_upload_handler.CreateSubmissionUploadHandler
	submit_flag_handler.SubmitFlagHandler
	list_my_submissions_handler.ListMySubmissionsHandler
	appeal_submission_handler.AppealSubmissionHandler
}

var _ flagsv1connect.FlagServiceHandler = FlagService{}

type AdminService struct {
	list_flags_handler.ListFlagsHandler
	create_flag_upload_handler.CreateFlagUploadHandler
	open_flag_handler.OpenFlagHandler
	add_time_handler.AddTimeHandler
	close_flag_handler.CloseFlagHandler
	list_submissions_handler.ListSubmissionsHandler
	approve_submission_handler.ApproveSubmissionHandler
	reject_submission_handler.RejectSubmissionHandler
	open_submitted_flag_handler.OpenSubmittedFlagHandler
	rename_flag_handler.RenameFlagHandler
}

var _ flagsv1connect.AdminServiceHandler = AdminService{}

type InternalService struct {
	get_payable_handler.GetPayableHandler
}

var _ flagsv1connect.InternalServiceHandler = InternalService{}
