import fs from "node:fs"
import path from "node:path"

import {latticeIds, latticeNeighbours} from "../../src/domain/lattice.ts"
import {cellOf, latticeFaces} from "../../src/domain/latticeCells.ts"
import {mapDir} from "./board.mjs"

// The carry-over between two boards: for each tile of the new board, the tile of the old board it comes from, or none,
// so a new coast or island starts blank. A switch of the world's board reads it once (see docs/boards.md, and
// map/README.md for the format).

export const carriesDir = path.join(mapDir, "carries")

// How far a new tile may lie from its parent on a coast, in the old board's spacings (docs/boards.md: about 0.6).
export const COAST_REACH = 0.6

const MAGIC = "CPCA"
const VERSION = 1
const HEADER = 24

/**
 * Each new tile's parent, as a 1-based tile id, or 0:
 * - the old tile whose lattice cell holds its position, the one a click there would have taken: inland every new tile
 *   has one, within 0.62 of the old spacing;
 * - where that cell is the sea, the nearest old tile among the cell's neighbours, when it lies within COAST_REACH
 *   spacings;
 * - otherwise none.
 * @param {{manifest: import("../../src/domain/board.ts").Board, count: number, ids: Uint32Array, points: Uint32Array,
 *   positions: Float32Array}} from
 * @param {{count: number, positions: Float32Array}} to
 * @returns {Uint32Array} one parent per new tile, in the new board's tile order
 */
export function parentsOf(from, to) {
    const shape = from.manifest.lattice
    const ids = latticeIds(shape)
    const pointOfId = new Uint32Array(ids.length)
    ids.forEach((id, point) => {
        pointOfId[id] = point
    })
    const tileOfPoint = new Int32Array(ids.length).fill(-1)
    for (let t = 0; t < from.count; t++) tileOfPoint[from.points[t]] = t
    const {at, to: next} = latticeNeighbours(shape)
    const reach = COAST_REACH * from.manifest.spacing
    const faces = latticeFaces(shape)
    const parents = new Uint32Array(to.count)
    for (let t = 0; t < to.count; t++) {
        const x = to.positions[t * 3], y = to.positions[t * 3 + 1], z = to.positions[t * 3 + 2]
        const point = pointOfId[cellOf(faces, shape, x, y, z)]
        const tile = tileOfPoint[point]
        if (tile >= 0) {
            parents[t] = tile + 1
            continue
        }
        let nearest = -1, least = reach
        for (let e = at[point]; e < at[point + 1]; e++) {
            const other = tileOfPoint[next[e]]
            if (other < 0) continue
            const length = arcTo(from.positions, other, x, y, z)
            if (length <= least) {
                least = length
                nearest = other
            }
        }
        parents[t] = nearest + 1
    }
    return parents
}

function arcTo(positions, tile, x, y, z) {
    const chord = Math.hypot(positions[tile * 3] - x, positions[tile * 3 + 1] - y, positions[tile * 3 + 2] - z)
    return 2 * Math.asin(Math.min(1, chord / 2))
}

/**
 * `"CPCA" | uint32 version | uint32 from's positionsHash | uint32 to's | uint32 from's tiles | uint32 to's | to's tiles ×
 * varint`: each parent less the one before it (0 before the first), zigzagged, seven bits a byte, low first.
 * @param {{positionsHash: number, tiles: number}} from the old board's manifest
 * @param {{positionsHash: number, tiles: number}} to the new board's manifest
 * @param {Uint32Array} parents
 * @returns {Uint8Array}
 */
export function encodeCarry(from, to, parents) {
    if (parents.length !== to.tiles) throw new Error(`${parents.length} parents for ${to.tiles} tiles`)
    const bytes = new Uint8Array(HEADER + parents.length * 5)
    const view = new DataView(bytes.buffer)
    for (let i = 0; i < 4; i++) bytes[i] = MAGIC.charCodeAt(i)
    view.setUint32(4, VERSION, true)
    view.setUint32(8, from.positionsHash, true)
    view.setUint32(12, to.positionsHash, true)
    view.setUint32(16, from.tiles, true)
    view.setUint32(20, to.tiles, true)
    let at = HEADER
    let before = 0
    for (const parent of parents) {
        if (parent > from.tiles) throw new Error(`parent ${parent} past the old board's ${from.tiles} tiles`)
        const delta = parent - before
        let zigzag = delta >= 0 ? delta * 2 : -delta * 2 - 1
        while (zigzag >= 0x80) {
            bytes[at++] = (zigzag & 0x7f) | 0x80
            zigzag = Math.floor(zigzag / 0x80)
        }
        bytes[at++] = zigzag
        before = parent
    }
    return bytes.slice(0, at)
}

/**
 * @param {Uint8Array} bytes
 * @returns {{fromHash: number, toHash: number, fromTiles: number, toTiles: number, parents: Uint32Array}}
 */
export function decodeCarry(bytes) {
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
    const magic = String.fromCharCode(...bytes.subarray(0, 4))
    if (magic !== MAGIC) throw new Error(`expected magic "${MAGIC}", got "${magic}"`)
    const version = view.getUint32(4, true)
    if (version !== VERSION) throw new Error(`unsupported carry version ${version}`)
    const fromTiles = view.getUint32(16, true), toTiles = view.getUint32(20, true)
    const parents = new Uint32Array(toTiles)
    let at = HEADER
    let before = 0
    for (let t = 0; t < toTiles; t++) {
        let zigzag = 0, scale = 1
        for (;;) {
            if (at >= bytes.length) throw new Error(`the file ends at tile ${t} of ${toTiles}`)
            const byte = bytes[at++]
            zigzag += (byte & 0x7f) * scale
            if (byte < 0x80) break
            scale *= 0x80
        }
        const parent = before + (zigzag % 2 === 0 ? zigzag / 2 : -(zigzag + 1) / 2)
        if (parent < 0 || parent > fromTiles) throw new Error(`tile ${t + 1}: parent ${parent} out of 0..${fromTiles}`)
        parents[t] = parent
        before = parent
    }
    if (at !== bytes.length) throw new Error(`${bytes.length - at} bytes after the last tile`)
    return {fromHash: view.getUint32(8, true), toHash: view.getUint32(12, true), fromTiles, toTiles, parents}
}

const carryPattern = (from, to) => new RegExp(`^carry-${from}-${to}-[0-9a-f]{8}\\.bin$`)

/** @returns {string | undefined} the carry-over's file from one board to another, if one is committed */
export function carryNamed(from, to) {
    if (!fs.existsSync(carriesDir)) return undefined
    const names = fs.readdirSync(carriesDir).filter((name) => carryPattern(from, to).test(name))
    if (names.length > 1) throw new Error(`expected one ${carriesDir}/carry-${from}-${to}-<hash>.bin, found ${names.length}`)
    return names[0]
}
