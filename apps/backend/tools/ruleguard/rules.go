//go:build ruleguard

package ruleguard

import "github.com/quasilyte/go-ruleguard/dsl"

func handRolledSet(m dsl.Matcher) {
	m.Match(`map[$_]struct{}`).
		Where(!m.File().PkgPath.Matches(`/shared/cpcolls$`)).
		Report(`use cpcolls.Set instead of a map to struct{}`)

	m.Match(`$m[$_] = true`).
		Where(m["m"].Type.Is(`map[$_]bool`)).
		Report(`use cpcolls.Set instead of a map to bool`)
}
