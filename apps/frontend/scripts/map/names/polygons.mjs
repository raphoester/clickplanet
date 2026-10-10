// Same seam rule as ground.mjs: a point exactly on ±180 misses every polygon.
const SEAM = 179.99
export const clampLon = (lon) => (lon > SEAM || lon < -SEAM ? (lon > 0 ? SEAM : -SEAM) : lon)

const GX = 360, GY = 180
const cellOf = (i, j) => j * GX + i

// Features may overlap: `contains` answers every one holding the point, `nearest` the closest within r degrees.
export function polygonIndex(features) {
    const shapes = []
    features.forEach((feature, f) => {
        const g = feature.geometry
        if (!g) return
        const polygons = g.type === "Polygon" ? [g.coordinates] : g.type === "MultiPolygon" ? g.coordinates : []
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
            shapes.push({f, rings, bbox: [minX, minY, maxX, maxY]})
        }
    })

    const grid = new Map()
    for (let s = 0; s < shapes.length; s++) {
        const [minX, minY, maxX, maxY] = shapes[s].bbox
        const j1 = Math.min(GY - 1, Math.floor(maxY + 90))
        const i1 = Math.min(GX - 1, Math.floor(maxX + 180))
        for (let j = Math.max(0, Math.floor(minY + 90)); j <= j1; j++) {
            for (let i = Math.max(0, Math.floor(minX + 180)); i <= i1; i++) {
                const k = cellOf(i, j)
                let bucket = grid.get(k)
                if (!bucket) grid.set(k, bucket = [])
                bucket.push(s)
            }
        }
    }

    const candidatesAround = (x, y, r) => {
        const out = new Set()
        const i0 = Math.max(0, Math.floor(x - r + 180)), i1 = Math.min(GX - 1, Math.floor(x + r + 180))
        const j0 = Math.max(0, Math.floor(y - r + 90)), j1 = Math.min(GY - 1, Math.floor(y + r + 90))
        for (let j = j0; j <= j1; j++) for (let i = i0; i <= i1; i++) for (const s of grid.get(cellOf(i, j)) ?? []) out.add(s)
        return out
    }

    return {
        contains(x, y) {
            const i = Math.min(GX - 1, Math.max(0, Math.floor(x + 180)))
            const j = Math.min(GY - 1, Math.max(0, Math.floor(y + 90)))
            const hits = []
            for (const s of grid.get(cellOf(i, j)) ?? []) {
                const {rings, bbox, f} = shapes[s]
                if (x < bbox[0] || x > bbox[2] || y < bbox[1] || y > bbox[3]) continue
                if (!inRing(rings[0], x, y)) continue
                let hole = false
                for (let r = 1; r < rings.length && !hole; r++) hole = inRing(rings[r], x, y)
                if (!hole && !hits.includes(f)) hits.push(f)
            }
            return hits
        },
        nearest(x, y, radius, accept = () => true) {
            let best = -1, bestD = radius
            const k = Math.cos(y * Math.PI / 180)
            const reach = radius / Math.max(0.05, k)
            for (const s of candidatesAround(x, y, reach)) {
                const {rings, bbox, f} = shapes[s]
                if (!accept(f)) continue
                if (x < bbox[0] - reach || x > bbox[2] + reach) continue
                if (y < bbox[1] - radius || y > bbox[3] + radius) continue
                for (const ring of rings) {
                    const d = ringDistance(ring, x, y, k)
                    if (d < bestD) {
                        bestD = d
                        best = f
                    }
                }
            }
            return best < 0 ? null : {f: best, d: bestD}
        },
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

// In degrees of latitude, longitude scaled by k = cos(lat).
function ringDistance(ring, x, y, k) {
    let best = Infinity
    const n = ring.length / 2
    for (let i = 0; i < n - 1; i++) {
        const ax = (ring[i * 2] - x) * k, ay = ring[i * 2 + 1] - y
        const bx = (ring[i * 2 + 2] - x) * k, by = ring[i * 2 + 3] - y
        const dx = bx - ax, dy = by - ay
        const len = dx * dx + dy * dy
        let t = len > 0 ? -(ax * dx + ay * dy) / len : 0
        t = Math.max(0, Math.min(1, t))
        const px = ax + t * dx, py = ay + t * dy
        const d = px * px + py * py
        if (d < best) best = d
    }
    return Math.sqrt(best)
}

export function pointDistance(lon1, lat1, lon2, lat2) {
    let dLon = Math.abs(lon1 - lon2)
    if (dLon > 180) dLon = 360 - dLon
    const k = Math.cos(((lat1 + lat2) / 2) * Math.PI / 180)
    return Math.hypot(dLon * k, lat1 - lat2)
}
