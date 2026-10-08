import {keyOf} from "./lattice.mjs"

// Across a cell edge the cover fades over this width, in units of the tile spacing.
const EDGE = 0.4

// Far enough that a pixel near any cell edge sees the vertices on both sides of it.
const REACH = 1.25

/**
 * @param {{count: number, positions: Float32Array}} vertices the whole lattice, not just the land
 * @param {Uint8Array} isTile 1 for each vertex that is a tile
 * @param {number} spacing the mean arc between two touching lattice vertices, from `spacingOf`
 * @param {number} width
 * @param {number} height
 * @returns {Float32Array} 1 where the nearest lattice vertex is a tile, 0 where it is sea
 */
export function coverage({count, positions}, isTile, spacing, width, height) {
    const nearestTile = new Float32Array(width * height).fill(Infinity)
    const nearestSea = new Float32Array(width * height).fill(Infinity)

    const columnX = new Float64Array(width)
    const columnZ = new Float64Array(width)
    for (let x = 0; x < width; x++) {
        const lon = ((x + 0.5) / width * 360 - 180) * Math.PI / 180
        columnX[x] = -Math.cos(lon)
        columnZ[x] = Math.sin(lon)
    }

    const rows = REACH * spacing * height / Math.PI

    for (let v = 0; v < count; v++) {
        const vx = positions[v * 3], vy = positions[v * 3 + 1], vz = positions[v * 3 + 2]
        const lat = Math.asin(Math.max(-1, Math.min(1, vy))) * 180 / Math.PI
        const lon = Math.atan2(vz, -vx) * 180 / Math.PI
        const cos = Math.max(Math.cos(lat * Math.PI / 180), 1e-4)
        const columns = Math.min(width / 2, rows / cos)

        const cx = (lon + 180) / 360 * width
        const cy = (90 - lat) / 180 * height

        const y0 = Math.max(0, Math.floor(cy - rows))
        const y1 = Math.min(height - 1, Math.ceil(cy + rows))
        const x0 = Math.floor(cx - columns)
        const x1 = Math.ceil(cx + columns)
        const nearest = isTile[v] ? nearestTile : nearestSea

        for (let y = y0; y <= y1; y++) {
            const pixelLat = (90 - (y + 0.5) / height * 180) * Math.PI / 180
            const py = Math.sin(pixelLat)
            const ring = Math.cos(pixelLat)
            const row = y * width
            for (let x = x0; x <= x1; x++) {
                const column = ((x % width) + width) % width
                const distance = Math.hypot(ring * columnX[column] - vx, py - vy, ring * columnZ[column] - vz)
                const at = row + column
                if (distance < nearest[at]) nearest[at] = distance
            }
        }
    }

    const cover = new Float32Array(width * height)
    const fade = 2 * EDGE * spacing
    for (let at = 0; at < cover.length; at++) {
        const tile = nearestTile[at], sea = nearestSea[at]
        if (sea === Infinity) cover[at] = tile === Infinity ? 0 : 1
        else if (tile !== Infinity) cover[at] = Math.min(1, Math.max(0, 0.5 + (sea - tile) / fade))
    }

    return cover
}

/**
 * @param {{count: number, positions: Float32Array}} vertices the whole lattice
 * @param {{count: number, positions: Float32Array}} tiles
 * @returns {Uint8Array} 1 for each lattice vertex that is a tile
 */
export function tilesOn(vertices, tiles) {
    const indexOf = new Map()
    for (let v = 0; v < vertices.count; v++) {
        indexOf.set(keyOf(vertices.positions[v * 3], vertices.positions[v * 3 + 1], vertices.positions[v * 3 + 2]), v)
    }
    const isTile = new Uint8Array(vertices.count)
    for (let t = 0; t < tiles.count; t++) {
        const v = indexOf.get(keyOf(tiles.positions[t * 3], tiles.positions[t * 3 + 1], tiles.positions[t * 3 + 2]))
        if (v === undefined) throw new Error(`tile ${t + 1} is not a lattice vertex`)
        isTile[v] = 1
    }
    return isTile
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
