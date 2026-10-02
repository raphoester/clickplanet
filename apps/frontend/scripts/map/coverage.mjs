import {lonLatOf} from "./lattice.mjs"

// Circumradius of a hexagonal cell, in units of the tile spacing.
const CELL = 1 / Math.sqrt(3)

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

                const at = row + ((x % width) + width) % width
                const value = distance <= inner ? 1 : (1 - distance) / (1 - inner)
                if (value > cover[at]) cover[at] = value
            }
        }
    }

    return cover
}

/**
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
