import {latticePositions, latticeSize, positionsHash, tilePositions} from "./lattice.ts"

export const LANDMASK_MAGIC = "CPLM"
export const LANDMASK_FORMAT_VERSION = 1
export const LANDMASK_HEADER_BYTES = 24

export type Landmask = {
    detail: number
    tiles: number
    hash: number
    isTile: Uint8Array
}

export class PositionsMismatchError extends Error {
    constructor(expected: number, got: number) {
        super(`rebuilt tile positions hash to ${hex(got)}, the landmask says ${hex(expected)}`)
        this.name = "PositionsMismatchError"
    }
}

// Tile positions must be the lattice's points, in its order; each tile is matched bit for bit.
export function landmaskOf(detail: number, positions: Float32Array): Landmask {
    const points = latticeSize(detail)
    const lattice = new Uint32Array(latticePositions(detail).buffer)
    const tiles = new Uint32Array(positions.buffer, positions.byteOffset, positions.length)
    const count = positions.length / 3

    const isTile = new Uint8Array(Math.ceil(points / 8))
    let tile = 0
    for (let point = 0; point < points && tile < count; point++) {
        if (lattice[point * 3] !== tiles[tile * 3]
            || lattice[point * 3 + 1] !== tiles[tile * 3 + 1]
            || lattice[point * 3 + 2] !== tiles[tile * 3 + 2]) continue
        isTile[point >> 3] |= 1 << (point & 7)
        tile++
    }
    if (tile !== count) throw new Error(`tile ${tile + 1} is not a point of the detail-${detail} lattice, in order`)

    return {detail, tiles: count, hash: positionsHash(positions), isTile}
}

export function encodeLandmask({detail, tiles, hash, isTile}: Landmask): ArrayBuffer {
    const points = latticeSize(detail)
    if (isTile.length !== Math.ceil(points / 8)) {
        throw new Error(`expected ${Math.ceil(points / 8)} bytes of bits, got ${isTile.length}`)
    }

    const buffer = new ArrayBuffer(LANDMASK_HEADER_BYTES + isTile.length)
    const view = new DataView(buffer)
    for (let i = 0; i < LANDMASK_MAGIC.length; i++) view.setUint8(i, LANDMASK_MAGIC.charCodeAt(i))
    view.setUint32(4, LANDMASK_FORMAT_VERSION, true)
    view.setUint32(8, detail, true)
    view.setUint32(12, points, true)
    view.setUint32(16, tiles, true)
    view.setUint32(20, hash, true)
    new Uint8Array(buffer, LANDMASK_HEADER_BYTES).set(isTile)
    return buffer
}

export function decodeLandmask(buffer: ArrayBuffer): Landmask {
    if (buffer.byteLength < LANDMASK_HEADER_BYTES) throw new Error(`landmask file is truncated: ${buffer.byteLength} bytes`)

    const view = new DataView(buffer)
    let magic = ""
    for (let i = 0; i < LANDMASK_MAGIC.length; i++) magic += String.fromCharCode(view.getUint8(i))
    if (magic !== LANDMASK_MAGIC) throw new Error(`not a landmask file: expected magic "${LANDMASK_MAGIC}", got "${magic}"`)

    const version = view.getUint32(4, true)
    if (version !== LANDMASK_FORMAT_VERSION) {
        throw new Error(`unsupported landmask format version ${version}, expected ${LANDMASK_FORMAT_VERSION}`)
    }

    const detail = view.getUint32(8, true)
    const points = view.getUint32(12, true)
    if (points !== latticeSize(detail)) throw new Error(`a detail-${detail} lattice has ${latticeSize(detail)} points, not ${points}`)

    const expectedBytes = LANDMASK_HEADER_BYTES + Math.ceil(points / 8)
    if (buffer.byteLength !== expectedBytes) {
        throw new Error(`landmask file size mismatch: ${points} points need ${expectedBytes} bytes, got ${buffer.byteLength}`)
    }

    return {
        detail,
        tiles: view.getUint32(16, true),
        hash: view.getUint32(20, true),
        isTile: new Uint8Array(buffer, LANDMASK_HEADER_BYTES),
    }
}

export function rebuildTilePositions(buffer: ArrayBuffer): Float32Array {
    const {detail, tiles, hash, isTile} = decodeLandmask(buffer)
    const positions = tilePositions(detail, isTile, tiles)
    const got = positionsHash(positions)
    if (got !== hash) throw new PositionsMismatchError(hash, got)
    return positions
}

function hex(hash: number): string {
    return hash.toString(16).padStart(8, "0")
}
