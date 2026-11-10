import {countryCodeOf} from "./ground.mjs"
import {positionOf} from "./seats.mjs"

const rankOf = (place) => {
    if (place.FEATURECLA === "Admin-0 capital" && place.ADM0CAP === 1 && place.CAPALT === 0) return 0
    return String(place.FEATURECLA).startsWith("Admin-0 ") ? 1 : 2
}

const before = (a, b) => a.rank - b.rank || Number(b.own) - Number(a.own) || b.pop - a.pop || a.name.localeCompare(b.name)

/**
 * The tile each country starts an arena on: the tile of its ground nearest its capital city. The city is its main
 * capital (`Admin-0 capital`, not the alternate one), else a territory's capital or a claimed one (`Admin-0 *`), else
 * its biggest place; among equals, a place of the country itself before one of a land it absorbs, then the biggest.
 * A country of one tile starts on it. A country with no place at all starts in the middle of its biggest landmass.
 * @param {{features: {properties: Record<string, unknown>}[]}} places Natural Earth's populated places
 * @param {{features: {properties: Record<string, unknown>}[]}} countries Natural Earth's countries, which name a place
 *   its own ISO code does not
 * @param {{codes: string[], landmass: Uint16Array, positions: Float32Array, frames: Float32Array, totals: Uint32Array}} map
 * @returns {Record<string, number>} each country's capital, as a wire tile id (index + 1)
 */
export function capitalsOf(places, countries, {codes, landmass, positions, frames, totals}) {
    const tilesOf = new Map()
    for (let t = 0; t < landmass.length; t++) {
        const code = codes[landmass[t]]
        if (!tilesOf.has(code)) tilesOf.set(code, [])
        tilesOf.get(code).push(t)
    }

    const absorbed = new Map()
    for (const feature of countries.features) {
        const code = countryCodeOf(feature)
        if (code) absorbed.set(feature.properties.ADM0_A3, code)
    }

    const cities = new Map()
    for (const {properties: place} of places.features) {
        const named = String(place.ISO_A2 ?? "").toLowerCase()
        const own = tilesOf.has(named)
        const code = own ? named : absorbed.get(place.ADM0_A3)
        if (!tilesOf.has(code)) continue
        const city = {
            rank: rankOf(place),
            own,
            pop: Number(place.POP_MAX ?? 0),
            name: String(place.NAME),
            at: positionOf([Number(place.LONGITUDE), Number(place.LATITUDE)]),
        }
        const best = cities.get(code)
        if (best === undefined || before(city, best) < 0) cities.set(code, city)
    }

    const capitals = {}
    for (const code of [...tilesOf.keys()].sort()) {
        const tiles = tilesOf.get(code)
        const at = tiles.length === 1 ? undefined : cities.get(code)?.at ?? middleOf(code, codes, frames, totals)
        capitals[code] = 1 + (at === undefined ? tiles[0] : nearest(positions, tiles, at))
    }
    return capitals
}

function middleOf(code, codes, frames, totals) {
    let biggest = -1
    for (let k = 1; k < codes.length; k++) {
        if (codes[k] === code && (biggest < 0 || totals[k] > totals[biggest])) biggest = k
    }
    return [frames[biggest * 5], frames[biggest * 5 + 1], frames[biggest * 5 + 2]]
}

function nearest(positions, tiles, [x, y, z]) {
    let best = tiles[0]
    let bestDistance = Infinity
    for (const t of tiles) {
        const d = (positions[t * 3] - x) ** 2 + (positions[t * 3 + 1] - y) ** 2 + (positions[t * 3 + 2] - z) ** 2
        if (d < bestDistance) {
            bestDistance = d
            best = t
        }
    }
    return best
}
