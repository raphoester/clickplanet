// A board's provinces, the ground fortify plays on (docs/provinces.md, /map/README.md#format): the province of each tile,
// and each province's landmass, frame and tile count. Province 0 is none: no tile holds it, and its row is zero.
export type Provinces = {
    // The borders file the landmasses are rows of: the 8 hex of its name, as a number.
    borders: number
    province: Uint16Array
    landmass: Uint16Array
    // Five floats a province, as the borders' frames: its centre x, y, z and its half extents east and north, in radians.
    frames: Float32Array
    tiles: Uint32Array
}

const MAGIC = "CPPV"
const VERSION = 1
const HEADER_BYTES = 20
const FRAME = 5

// The borders file a board's provinces were cut on, by its name: "borders-0ae1a037.bin" is 0x0ae1a037.
export function bordersHashOf(name: string): number {
    const hash = /^borders-([0-9a-f]{8})\.bin$/.exec(name)?.[1]
    if (hash === undefined) throw new Error(`${name} is not a borders file's name`)
    return parseInt(hash, 16)
}

export function encodeProvinces({borders, province, landmass, frames, tiles}: Provinces): Uint8Array {
    const count = landmass.length
    if (tiles.length !== count || frames.length !== count * FRAME) {
        throw new Error(`${count} provinces, ${tiles.length} counts and ${frames.length / FRAME} frames`)
    }
    const bytes = new Uint8Array(HEADER_BYTES + count * (FRAME * 4 + 4 + 2) + province.length * 2)
    const view = new DataView(bytes.buffer)
    for (let i = 0; i < MAGIC.length; i++) view.setUint8(i, MAGIC.charCodeAt(i))
    view.setUint32(4, VERSION, true)
    view.setUint32(8, province.length, true)
    view.setUint32(12, count, true)
    view.setUint32(16, borders, true)
    let at = HEADER_BYTES
    for (const value of frames) {
        view.setFloat32(at, value, true)
        at += 4
    }
    for (const value of tiles) {
        view.setUint32(at, value, true)
        at += 4
    }
    for (const value of landmass) {
        view.setUint16(at, value, true)
        at += 2
    }
    for (const value of province) {
        if (value === 0 || value >= count) throw new Error(`a tile in province ${value}, of 1 to ${count - 1}`)
        view.setUint16(at, value, true)
        at += 2
    }
    return bytes
}

export function decodeProvinces(buffer: ArrayBuffer, tileCount: number): Provinces {
    if (buffer.byteLength < HEADER_BYTES) throw new Error(`provinces file is truncated: ${buffer.byteLength} bytes`)
    const view = new DataView(buffer)
    let magic = ""
    for (let i = 0; i < MAGIC.length; i++) magic += String.fromCharCode(view.getUint8(i))
    if (magic !== MAGIC) throw new Error(`not a provinces file: expected magic "${MAGIC}", got "${magic}"`)

    const version = view.getUint32(4, true)
    if (version !== VERSION) throw new Error(`unsupported provinces format version ${version}, expected ${VERSION}`)
    const held = view.getUint32(8, true)
    if (held !== tileCount) throw new Error(`the provinces are of ${held} tiles, the board has ${tileCount}`)
    const count = view.getUint32(12, true)
    const expected = HEADER_BYTES + count * (FRAME * 4 + 4 + 2) + held * 2
    if (buffer.byteLength !== expected) {
        throw new Error(`provinces file size mismatch: ${count} provinces of ${held} tiles need ${expected} bytes, got ${buffer.byteLength}`)
    }

    let at = HEADER_BYTES
    const frames = new Float32Array(buffer, at, count * FRAME)
    at += count * FRAME * 4
    const tiles = new Uint32Array(buffer, at, count)
    at += count * 4
    const landmass = new Uint16Array(buffer, at, count)
    at += count * 2
    const province = new Uint16Array(buffer, at, held)
    for (let t = 0; t < held; t++) {
        if (province[t] === 0 || province[t] >= count) throw new Error(`tile ${t + 1} is in province ${province[t]}, of 1 to ${count - 1}`)
    }
    return {borders: view.getUint32(16, true), province, landmass, frames, tiles}
}
