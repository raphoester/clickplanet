import {latticeNeighbours, latticePositions, latticeSize} from "../../src/domain/lattice.ts"

// The remap follows a tile from one tile set to the next by its position, written so.
export const keyOf = (x, y, z) => `${x.toFixed(6)},${y.toFixed(6)},${z.toFixed(6)}`

/**
 * @param {number} cols
 * @returns {{count: number, positions: Float32Array}} every point of the lattice, in three's order
 */
export function lattice(cols) {
    return {count: latticeSize(cols), positions: latticePositions(cols)}
}

/**
 * @param {number} cols
 * @returns {{count: number, at: Uint32Array, to: Uint32Array}} CSR: `at[i]` to `at[i + 1]` indexes `to`, ascending
 */
export function neighbours(cols) {
    return {count: latticeSize(cols), ...latticeNeighbours(cols)}
}

/**
 * Latitude is asin(y) and longitude atan2(z, −x), in degrees.
 * @param {ArrayLike<number>} positions
 * @param {number} v
 * @returns {[number, number]} [lon, lat]
 */
export function lonLatOf(positions, v) {
    const x = positions[v * 3], y = positions[v * 3 + 1], z = positions[v * 3 + 2]
    return [Math.atan2(z, -x) * 180 / Math.PI, Math.asin(Math.max(-1, Math.min(1, y))) * 180 / Math.PI]
}
