package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/drop_bomb_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/map_density_handler"
)

// ClickService exists because the generated handler wants one value carrying all
// six procedures. That is the whole of its job, and it has no constructor
// because there is nothing to construct: it is six handlers in a bag, and the
// DI sequence that already builds them writes the literal.
//
// Each procedure is a package of its own, holding the one use case it calls and
// declaring the one port it needs. What GetMap reads the map with is therefore
// not merely unused by Click, it is unreachable from it — which is what stops
// this drifting back into one struct that grows a field per feature.
//
// The six are embedded, so every method is promoted rather than written: there
// is no delegation to keep in step with the generated interface, and no test to
// write here either — an aggregation's only claim is the assertion below.
type ClickService struct {
	click_handler.ClickHandler
	get_budget_handler.GetBudgetHandler
	map_density_handler.MapDensityHandler
	get_map_handler.GetMapHandler
	listen_for_events_handler.ListenForEventsHandler
	claim_bonus_handler.ClaimBonusHandler
	drop_bomb_handler.DropBombHandler
}

var _ planetv1connect.ClickServiceHandler = ClickService{}
