import {latticeNeighbours, latticePositions, latticeSize} from "../../src/domain/lattice.ts"

// The remap follows a tile from one tile set to the next by its position, written so.
export const keyOf = (x, y, z) => `${x.toFixed(6)},${y.toFixed(6)},${z.toFixed(6)}`

/**
 * @param {import("../../src/domain/lattice.ts").Lattice | number} given a board's lattice, or `cols` for three's
 * @returns {{count: number, positions: Float32Array}} every point of the lattice, in its order (three's or canonical)
 */
export function lattice(given) {
    const cols = typeof given === "number" ? given : given.cols
    return {count: latticeSize(cols), positions: latticePositions(given)}
}

/**
 * @param {import("../../src/domain/lattice.ts").Lattice | number} given a board's lattice, or `cols` for three's
 * @returns {{count: number, at: Uint32Array, to: Uint32Array}} CSR: `at[i]` to `at[i + 1]` indexes `to`, ascending
 */
export function neighbours(given) {
    const cols = typeof given === "number" ? given : given.cols
    return {count: latticeSize(cols), ...latticeNeighbours(given)}
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
