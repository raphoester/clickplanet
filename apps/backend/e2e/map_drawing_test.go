package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
)

func (s gameStack) mapDesk() planetv1connect.AdminServiceClient {
	return planetv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL)
}

func (s gameStack) restoreArea(t *testing.T, tile uint32, before time.Time, dryRun bool) *planetv1.RestoreAreaResponse {
	t.Helper()

	res, err := s.mapDesk().RestoreArea(t.Context(), connect.NewRequest(&planetv1.RestoreAreaRequest{
		TileId: tile, Rings: 1, Before: timestamppb.New(before), DryRun: dryRun,
	}))
	require.NoError(t, err)
	return res.Msg
}

func (s gameStack) clearArea(t *testing.T, tile uint32, dryRun bool) *planetv1.ClearAreaResponse {
	t.Helper()

	res, err := s.mapDesk().ClearArea(t.Context(), connect.NewRequest(&planetv1.ClearAreaRequest{
		TileId: tile, Rings: 1, DryRun: dryRun,
	}))
	require.NoError(t, err)
	return res.Msg
}

func (p *gamer) reportPlace(tile uint32, reason supportv1.PlaceReportReason, text string) error {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.ReportPlaceRequest{TileId: tile, Reason: reason, Text: text})
	p.send(req.Header())
	_, err := p.support().ReportPlace(p.t.Context(), req)
	return err //nolint:wrapcheck // the tests read the connect code.
}

func (s gameStack) placeReports(t *testing.T, openOnly bool) []*supportv1.PlaceReport {
	t.Helper()

	res, err := s.supportDesk().ListPlaceReports(t.Context(),
		connect.NewRequest(&supportv1.ListPlaceReportsRequest{OpenOnly: openOnly}))
	require.NoError(t, err)
	return res.Msg.GetReports()
}

func TestAnOperatorSeesWhoDrewAnAreaAndPutsItBackAsItWas(t *testing.T) {
	game := startGame(t)
	ada, drawer := game.newPlayer(t), game.newPlayer(t)
	ada.click(1, "fr")
	before := time.Now().Truncate(time.Second).Add(time.Second)
	time.Sleep(time.Until(before))
	drawer.click(1, "de")

	seen := game.restoreArea(t, 1, before, true)
	assert.Positive(t, seen.GetTiles())
	assert.Equal(t, uint32(1), seen.GetChanged())
	assert.Zero(t, seen.GetRestored(), "a dry run restores nothing")
	require.Len(t, seen.GetPainters(), 1, "the take before the time is not the drawing")
	assert.Equal(t, drawer.account(), seen.GetPainters()[0].GetAccountId())
	assert.Equal(t, "de", seen.GetPainters()[0].GetFlagCountryId())

	restored := game.restoreArea(t, 1, before, false)
	assert.Equal(t, uint32(1), restored.GetRestored())
	assert.Zero(t, game.restoreArea(t, 1, before, true).GetChanged(), "the area holds what it held then")
	assert.Equal(t, uint32(1), game.clearArea(t, 1, true).GetOwned(), "fr holds its tile again")

	_, err := game.mapDesk().RestoreArea(t.Context(), connect.NewRequest(&planetv1.RestoreAreaRequest{
		TileId: 1, Rings: 1, Before: timestamppb.New(time.Now().Add(-73 * time.Hour)),
	}))
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "past what the ledger keeps")
}

func TestAnOperatorClearsAnAreaTooOldForTheLedger(t *testing.T) {
	game := startGame(t)
	game.newPlayer(t).click(1, "de")

	assert.Equal(t, uint32(1), game.clearArea(t, 1, true).GetOwned())
	assert.Equal(t, uint32(1), game.clearArea(t, 1, false).GetCleared())
	assert.Zero(t, game.clearArea(t, 1, true).GetOwned())

	_, err := game.mapDesk().ClearArea(t.Context(), connect.NewRequest(&planetv1.ClearAreaRequest{TileId: 1, Rings: 51}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAGuestReportsADrawingAndTheOperatorClosesTheReport(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)
	said, err := guest.post()
	require.NoError(t, err)

	require.NoError(t, guest.reportPlace(42, supportv1.PlaceReportReason_PLACE_REPORT_REASON_HATE, "  a swastika  "))

	open := game.placeReports(t, true)
	require.Len(t, open, 1)
	assert.Equal(t, guest.account(), open[0].GetReporterAccountId())
	assert.Equal(t, said.GetAuthorName(), open[0].GetReporterName())
	assert.Equal(t, uint32(42), open[0].GetTileId())
	assert.Equal(t, supportv1.PlaceReportReason_PLACE_REPORT_REASON_HATE, open[0].GetReason())
	assert.Equal(t, "a swastika", open[0].GetText())

	_, err = game.supportDesk().DecidePlaceReport(t.Context(), connect.NewRequest(&supportv1.DecidePlaceReportRequest{
		ReportId: open[0].GetId(), Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_ACTED, Comment: "restored the area",
	}))
	require.NoError(t, err)

	assert.Empty(t, game.placeReports(t, true))
	all := game.placeReports(t, false)
	require.Len(t, all, 1)
	assert.Equal(t, supportv1.ReportOutcome_REPORT_OUTCOME_ACTED, all[0].GetDecision().GetOutcome())
}

func TestAPlaceReportNeedsATileAndAReasonAndAnAccount(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	err := guest.reportPlace(0, supportv1.PlaceReportReason_PLACE_REPORT_REASON_HATE, "")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "no tile")

	err = guest.reportPlace(42, supportv1.PlaceReportReason_PLACE_REPORT_REASON_UNSPECIFIED, "")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "no reason")

	nobody := &gamer{t: t, stack: game}
	err = nobody.reportPlace(42, supportv1.PlaceReportReason_PLACE_REPORT_REASON_HATE, "")
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	assert.Empty(t, game.placeReports(t, false))
}

func TestADeletedAccountsPlaceReportsStayWithNoReporter(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)
	require.NoError(t, guest.reportPlace(42, supportv1.PlaceReportReason_PLACE_REPORT_REASON_OTHER, "an ad"))

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	guest.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		reports := game.placeReports(t, false)
		return len(reports) == 1 && reports[0].GetReporterAccountId() == ""
	}, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, "an ad", game.placeReports(t, false)[0].GetText(), "the notice stays")
}
