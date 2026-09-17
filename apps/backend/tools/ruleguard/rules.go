//go:build ruleguard

// Package ruleguard holds the project's own lint rules, run by gocritic's
// ruleguard check. See .golangci.yaml.
package ruleguard

import "github.com/quasilyte/go-ruleguard/dsl"

// handRolledSet catches a set built on a map: cpcolls.Set says what it is and
// owns the idiom, so no call site writes `_, ok := m[k]` for "contains" again.
func handRolledSet(m dsl.Matcher) {
	m.Match(`map[$_]struct{}`).
		Where(!m.File().PkgPath.Matches(`/shared/cpcolls$`)).
		Report(`use cpcolls.Set instead of a map to struct{}`)

	m.Match(`$m[$_] = true`).
		Where(m["m"].Type.Is(`map[$_]bool`)).
		Report(`use cpcolls.Set instead of a map to bool`)
}
