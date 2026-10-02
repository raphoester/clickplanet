import {naturalEarth} from "./naturalEarth.mjs"

export const SEA = ""

const ABSORBED = {SOL: "so", CYN: "cy"}

const ANTARCTICA = "aq"

// Natural Earth cuts Antarctica at ±180, and a point exactly on the cut misses every polygon.
const SEAM = 179.99

/**
 * @param {{properties: Record<string, unknown>}} feature
 * @returns {string | null}
 */
export function countryCodeOf(feature) {
    const named = String(feature.properties.ISO_A2_EH ?? feature.properties.ISO_A2 ?? "").toLowerCase()
    const code = named && named !== "-99" ? named : ABSORBED[feature.properties.ADM0_A3] ?? ""
    return code || null
}

/**
 * @param {{countries: {features: object[]}, iceShelves: {features: object[]}}} datasets
 * @returns {{groundOf: (lon: number, lat: number) => string}}
 */
export function groundIndex({countries, iceShelves}) {
    const layers = [
        shapesOf(countries.features, countryCodeOf),
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

export async function loadGround() {
    const [countries, iceShelves] = await Promise.all([
        naturalEarth("ne_50m_admin_0_countries"),
        naturalEarth("ne_50m_antarctic_ice_shelves_polys"),
    ])
    return groundIndex({countries, iceShelves})
}

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
