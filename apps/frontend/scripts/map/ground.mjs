// The one answer to "what is under this point".
//
// Where a tile is and who owns the ground under it used to be two questions asked of two different
// datasets — a greyscale land mask decided which lattice vertices became tiles, and Natural Earth
// decided which country each tile sat in. They disagreed on about 6,000 vertices, all of them on
// coasts and islands, which is exactly where anyone looks: tiles floating on open water with no
// border round them, islands drawn in the texture with nothing to click, and the Antarctic ice
// shelves missing whole.
//
// So there is one oracle now, and both questions are the same call. `groundOf` answers a country
// code or SEA, and a tile exists exactly where it answers a code. A tile with no country is no
// longer a case the rest of the pipeline has to have an answer for: it cannot be built.
//
// Measured against the union of the country polygons, `ne_50m_land` is the same shape to within 4
// lattice vertices (the Siachen Glacier, which India and Pakistan both claim and which Natural Earth
// declines to hand to either). So the country file answers sea/land too and there is no second
// coastline to keep in step.
import {naturalEarth} from "./naturalEarth.mjs"

export const SEA = ""

// Natural Earth leaves ISO_A2 at -99 for the three territories it maps but does not hand a country
// code to. The flag layer paints per piece of land, so a territory without a code is not a blank
// flag, it is a hole: 327 tiles of Somaliland showed bare ground while the tiles under them were
// owned and wearing a flag up close. Fold the two that sit inside a country into it — Natural
// Earth's own ADM0_ISO says which. Siachen has no ADM0_ISO either, so its 4 vertices stay sea.
const ABSORBED = {SOL: "so", CYN: "cy"}

// The ice shelves are a separate Natural Earth file and are in no country polygon, which is why
// Ross and Ronne had no tiles on them while the globe texture painted them solid white. They are
// Antarctic by definition — `iceIsAntarctic` in the test pins that against the data.
const ANTARCTICA = "aq"

// A country that wraps the globe cannot be a ring in longitude and latitude without being cut
// somewhere, and Natural Earth cuts Antarctica down the antimeridian. A point landing exactly on
// 180 therefore lands exactly on the polygon's own edge, where an even-odd ray test has no answer
// to give, and falls out of every country — 24 of them, in a row, straight out from the south pole.
// On the globe they were the one line of ground not wearing the flag. Nudging off the cut costs a
// hundredth of a degree, about a kilometre.
const SEAM = 179.99

/**
 * Builds the oracle from already-loaded geojson. Pure, so the rules are testable without a network.
 *
 * @param {{countries: {features: object[]}, iceShelves: {features: object[]}}} datasets
 * @returns {{groundOf: (lon: number, lat: number) => string}}
 */
export function groundIndex({countries, iceShelves}) {
    // Country polygons are asked first so that a shelf overlapping a claimed coast reads as the
    // country rather than as bare Antarctica.
    const layers = [
        shapesOf(countries.features, (f) => {
            const named = (f.properties.ISO_A2_EH ?? f.properties.ISO_A2 ?? "").toLowerCase()
            const code = named && named !== "-99" ? named : ABSORBED[f.properties.ADM0_A3] ?? ""
            return code || null
        }),
        shapesOf(iceShelves.features, () => ANTARCTICA),
    ]

    return {
        groundOf(lon, lat) {
            const x = lon > SEAM || lon < -SEAM ? (lon > 0 ? SEAM : -SEAM) : lon
            for (const layer of layers) {
                const code = layer(x, lat)
                if (code !== null) return code
            }
            return SEA
        },
    }
}

/** Fetches the pinned datasets and builds the oracle. */
export async function loadGround() {
    const [countries, iceShelves] = await Promise.all([
        naturalEarth("ne_50m_admin_0_countries"),
        naturalEarth("ne_50m_antarctic_ice_shelves_polys"),
    ])
    return groundIndex({countries, iceShelves})
}

// --- one layer: polygons flattened to rings, indexed on a 1-degree grid.

const GX = 360, GY = 180
const cell = (i, j) => j * GX + i

function shapesOf(features, codeOf) {
    const shapes = []
    for (const feature of features) {
        const code = codeOf(feature)
        if (code === null) continue
        const polygons = feature.geometry.type === "Polygon"
            ? [feature.geometry.coordinates]
            : feature.geometry.coordinates
        for (const polygon of polygons) {
            let minX = 180, minY = 90, maxX = -180, maxY = -90
            const rings = polygon.map((ring) => {
                const flat = new Float64Array(ring.length * 2)
                for (let i = 0; i < ring.length; i++) {
                    const [x, y] = ring[i]
                    flat[i * 2] = x
                    flat[i * 2 + 1] = y
                    if (x < minX) minX = x
                    if (x > maxX) maxX = x
                    if (y < minY) minY = y
                    if (y > maxY) maxY = y
                }
                return flat
            })
            shapes.push({code, rings, bbox: [minX, minY, maxX, maxY]})
        }
    }

    const grid = new Map()
    for (let s = 0; s < shapes.length; s++) {
        const [minX, minY, maxX, maxY] = shapes[s].bbox
        const j1 = Math.min(GY - 1, Math.floor(maxY + 90))
        const i1 = Math.min(GX - 1, Math.floor(maxX + 180))
        for (let j = Math.max(0, Math.floor(minY + 90)); j <= j1; j++) {
            for (let i = Math.max(0, Math.floor(minX + 180)); i <= i1; i++) {
                const k = cell(i, j)
                let bucket = grid.get(k)
                if (!bucket) grid.set(k, bucket = [])
                bucket.push(s)
            }
        }
    }

    return (x, y) => {
        const i = Math.min(GX - 1, Math.max(0, Math.floor(x + 180)))
        const j = Math.min(GY - 1, Math.max(0, Math.floor(y + 90)))
        const candidates = grid.get(cell(i, j))
        if (!candidates) return null
        for (const s of candidates) {
            const {rings, bbox, code} = shapes[s]
            if (x < bbox[0] || x > bbox[2] || y < bbox[1] || y > bbox[3]) continue
            if (!inRing(rings[0], x, y)) continue
            let hole = false
            for (let r = 1; r < rings.length && !hole; r++) hole = inRing(rings[r], x, y)
            if (!hole) return code
        }
        return null
    }
}

function inRing(ring, x, y) {
    let inside = false
    const n = ring.length / 2
    for (let i = 0, j = n - 1; i < n; j = i++) {
        const xi = ring[i * 2], yi = ring[i * 2 + 1]
        const xj = ring[j * 2], yj = ring[j * 2 + 1]
        if ((yi > y) !== (yj > y) && x < (xj - xi) * (y - yi) / (yj - yi) + xi) inside = !inside
    }
    return inside
}
