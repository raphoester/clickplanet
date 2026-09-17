// Package playerv1controller is the player module's edge: player.v1.PlayerService for players, and
// player.v1.InternalService for the other modules.
package playerv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_author_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
)

// PlayerService is the three handlers in a bag, for the generated handler.
type PlayerService struct {
	get_profile_handler.GetProfileHandler
	set_name_handler.SetNameHandler
	get_stats_handler.GetStatsHandler
}

var _ playerv1connect.PlayerServiceHandler = PlayerService{}

// InternalService is what the other modules ask, served on the internal listener alone.
type InternalService struct {
	get_author_handler.GetAuthorHandler
}

var _ playerv1connect.InternalServiceHandler = InternalService{}
