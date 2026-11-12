import {DETAIL, keyOf, lattice, neighbours} from "./lattice.mjs"

/**
 * The lattice under a map's tiles: its edges by vertex, and each vertex's tile index, or -1 at sea.
 * @param {ArrayLike<number>} positions the tiles' positions, x, y, z per tile
 * @returns {{at: Uint32Array, to: Uint32Array, tileOf: Int32Array}}
 */
export function tileLattice(positions) {
    const tileAt = new Map()
    for (let t = 0; t < positions.length / 3; t++) {
        tileAt.set(keyOf(positions[t * 3], positions[t * 3 + 1], positions[t * 3 + 2]), t)
    }
    const {count, positions: vertices} = lattice(DETAIL)
    const tileOf = new Int32Array(count)
    for (let v = 0; v < count; v++) {
        tileOf[v] = tileAt.get(keyOf(vertices[v * 3], vertices[v * 3 + 1], vertices[v * 3 + 2])) ?? -1
    }
    const {at, to} = neighbours(DETAIL)
    return {at, to, tileOf}
}
