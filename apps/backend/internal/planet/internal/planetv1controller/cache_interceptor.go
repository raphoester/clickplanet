package planetv1controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

const mapMaxAge = 5

const rulesMaxAge = 300

type MapVersions interface {
	Version() clicks.MapVersion
}

type versioned interface {
	GetVersion() *planetv1.MapVersion
}

func NewCacheInterceptor(versions MapVersions) connect.Interceptor {
	maxAge := map[string]int{
		planetv1connect.ClickServiceGetMapProcedure:        mapMaxAge,
		planetv1connect.ClickServiceGetHoldingsProcedure:   mapMaxAge,
		planetv1connect.ClickServiceMapDensityProcedure:    mapMaxAge,
		planetv1connect.ClickServiceGetBonusRulesProcedure: rulesMaxAge,
		planetv1connect.ClickServiceGetFortressesProcedure: mapMaxAge,
		planetv1connect.ClickServiceGetLandmassProcedure:   mapMaxAge,
	}
	tagged := map[string]bool{
		planetv1connect.ClickServiceGetMapProcedure:      true,
		planetv1connect.ClickServiceGetHoldingsProcedure: true,
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure
			age, cacheable := maxAge[procedure]
			if !cacheable {
				return next(ctx, req)
			}
			cacheControl := fmt.Sprintf("public, max-age=%d, s-maxage=%d", age, age)

			if tagged[procedure] && req.HTTPMethod() == http.MethodGet {
				if tag := entityTag(planetmessage.Version(versions.Version())); matches(req.Header().Get("If-None-Match"), tag) {
					return nil, connect.NewNotModifiedError(http.Header{"Etag": {tag}, "Cache-Control": {cacheControl}})
				}
			}

			res, err := next(ctx, req)
			if err != nil {
				return res, err
			}

			res.Header().Set("Cache-Control", cacheControl)
			if message, ok := res.Any().(versioned); ok && tagged[procedure] {
				res.Header().Set("Etag", entityTag(message.GetVersion()))
			}

			return res, nil
		}
	})
}

func entityTag(version *planetv1.MapVersion) string {
	return fmt.Sprintf(`"%d.%d"`, version.GetEpoch(), version.GetSeq())
}

// Weak comparison, as RFC 9110 asks of If-None-Match: Cloudflare weakens a tag when it compresses.
func matches(ifNoneMatch, tag string) bool {
	for candidate := range strings.SplitSeq(ifNoneMatch, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == tag || candidate == "*" {
			return true
		}
	}
	return false
}
