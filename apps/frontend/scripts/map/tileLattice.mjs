import {tileOfPoint} from "./board.mjs"
import {neighbours} from "./lattice.mjs"

/**
 * The lattice under a board's tiles: its edges by point, and each point's tile index, or -1 at sea.
 * @param {{count: number, points: Uint32Array, manifest: {lattice: {cols: number}}}} board
 * @returns {{at: Uint32Array, to: Uint32Array, tileOf: Int32Array}}
 */
export function tileLattice(board) {
    const {count, at, to} = neighbours(board.manifest.lattice.cols)
    return {at, to, tileOf: tileOfPoint(board, count)}
}
