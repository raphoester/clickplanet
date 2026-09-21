// Where the tile field covers the globe, rasterised onto an equirectangular image.
//
// A tile's ground is its Voronoi cell on the geodesic lattice — a hexagon whose circumradius is the
// spacing over root 3, which is the same 1.155x the renderer widens the discs by when it needs them
// to cover the ground for the painted flag. So "is there a tile here" has an answer for every point,
// not just for the lattice vertices, and it is the answer the player can act on: inside a cell there
// is a disc to click, outside it there is none.
import {lonLatOf} from "./lattice.mjs"

// The circumradius of a hexagonal cell, as a multiple of the spacing between two touching tiles.
// Below it the cells leave holes between them; above it they overlap into the sea.
const CELL = 1 / Math.sqrt(3)

// How far past the cell the coverage fades to nothing, again as a multiple of the spacing. A hard
// edge on a 4096x2048 image is a staircase at the zoom this texture is looked at.
const FEATHER = 0.25

/**
 * @param {{count: number, uvs: Float32Array}} tiles
 * @param {number} spacing the mean arc between two touching lattice vertices, from `spacingOf`
 * @param {number} width
 * @param {number} height
 * @returns {Float32Array} 1 on a tile's own ground, 0 at sea
 */
export function coverage(tiles, spacing, width, height) {
    const cover = new Float32Array(width * height)

    // Degrees, and then pixels. The image is equirectangular, so a degree of latitude is always the
    // same number of rows, and a degree of longitude is always the same number of columns — but the
    // *ground* a column covers shrinks with the cosine of the latitude, which is why the ellipse
    // painted per tile is as wide as the latitude makes it.
    const radius = (CELL + FEATHER) * spacing * 180 / Math.PI
    const rows = radius * height / 180
    const inner = CELL / (CELL + FEATHER)

    for (let t = 0; t < tiles.count; t++) {
        const [lon, lat] = lonLatOf(tiles.uvs[t * 2], tiles.uvs[t * 2 + 1])
        const cos = Math.max(Math.cos(lat * Math.PI / 180), 1e-4)
        const columns = Math.min(width / 2, rows / cos)

        const cx = (lon + 180) / 360 * width
        const cy = (90 - lat) / 180 * height

        const y0 = Math.max(0, Math.floor(cy - rows))
        const y1 = Math.min(height - 1, Math.ceil(cy + rows))
        const x0 = Math.floor(cx - columns)
        const x1 = Math.ceil(cx + columns)

        for (let y = y0; y <= y1; y++) {
            const dy = (y + 0.5 - cy) / rows
            const row = y * width
            for (let x = x0; x <= x1; x++) {
                const dx = (x + 0.5 - cx) / columns
                const distance = Math.hypot(dx, dy)
                if (distance >= 1) continue

                // The image wraps in longitude, so a tile near the seam paints into both edges.
                const at = row + ((x % width) + width) % width
                const value = distance <= inner ? 1 : (1 - distance) / (1 - inner)
                if (value > cover[at]) cover[at] = value
            }
        }
    }

    return cover
}

/**
 * The mean arc between two touching lattice vertices, measured rather than configured: it is a
 * property of the lattice the blob was cut from, and a number in a config beside it could drift
 * from it. The backend measures the same thing at boot, for the bomb's radius.
 *
 * Measured over the edges themselves, not over "the nearest other tile": the spacing varies about
 * 25% between a face's middle and its corners, and a nearest-neighbour search over an array in
 * generation order quietly answers a different question at every coastline.
 *
 * @param {Float32Array} positions the whole lattice, not just the land
 * @param {{at: Uint32Array, to: Uint32Array, count: number}} edges
 * @returns {number} radians
 */
export function spacingOf(positions, {at, to, count}) {
    let total = 0
    let taken = 0
    for (let v = 0; v < count; v++) {
        for (let e = at[v]; e < at[v + 1]; e++) {
            const other = to[e]
            if (other < v) continue
            const dx = positions[v * 3] - positions[other * 3]
            const dy = positions[v * 3 + 1] - positions[other * 3 + 1]
            const dz = positions[v * 3 + 2] - positions[other * 3 + 2]
            total += 2 * Math.asin(Math.hypot(dx, dy, dz) / 2)
            taken++
        }
    }
    return total / taken
}
