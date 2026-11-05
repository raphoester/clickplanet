package antibotv1controller

import (
	"net/http"
	"net/url"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

// Deprecated: until the backoffice calls antibot.v1; each message reads on the wire as its planet.v1 twin.
func PlanetAliases(service AdminService) []cpbootstrap.ServiceBuilder {
	return []cpbootstrap.ServiceBuilder{
		alias(service, planetv1connect.AdminServiceBanPlayerProcedure, antibotv1connect.AdminServiceBanPlayerProcedure),
		alias(service, planetv1connect.AdminServiceUnbanPlayerProcedure, antibotv1connect.AdminServiceUnbanPlayerProcedure),
		alias(service, planetv1connect.AdminServiceInspectPlayerProcedure, antibotv1connect.AdminServiceInspectPlayerProcedure),
	}
}

func alias(service AdminService, old, procedure string) cpbootstrap.ServiceBuilder {
	return func(options ...connect.HandlerOption) (string, http.Handler) {
		_, handler := antibotv1connect.NewAdminServiceHandler(service, options...)

		return old, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			moved := r.Clone(r.Context())
			moved.URL = &url.URL{Path: procedure, RawQuery: r.URL.RawQuery}
			handler.ServeHTTP(w, moved)
		})
	}
}
