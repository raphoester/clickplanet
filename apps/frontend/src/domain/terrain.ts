// What each point of a board's lattice looks like, one kind a point by canonical id (/map/README.md#format): under a
// tile, a kind of land; at sea, how many lattice steps it is from land, four and more being the open sea.
export const LAND = ["forest", "tundra", "boreal", "arid", "savanna", "grass", "sand", "snow"] as const
export const SEA_BANDS = 4

export type Land = typeof LAND[number]

export const KINDS = LAND.length + SEA_BANDS
export const OPEN_SEA = KINDS - 1

// The kind of a sea point `steps` from land.
export const seaKind = (steps: number): number => LAND.length + Math.min(steps, SEA_BANDS) - 1

const MAGIC = "CPTR"
const VERSION = 1
const HEADER_BYTES = 12
// A run's length minus one fits the byte's low four bits up to this, after which the rest follows as a varint.
const LONG = 15

export function encodeTerrain(kinds: Uint8Array): Uint8Array {
    const out: number[] = []
    for (let at = 0; at < kinds.length;) {
        const kind = kinds[at]
        if (kind >= KINDS) throw new Error(`point ${at} is kind ${kind}, past the ${KINDS} kinds`)
        let end = at + 1
        while (end < kinds.length && kinds[end] === kind) end++
        const length = end - at
        out.push(kind << 4 | Math.min(length - 1, LONG))
        if (length - 1 >= LONG) {
            for (let rest = length - 1 - LONG; ; rest = Math.floor(rest / 128)) {
                out.push(rest >= 128 ? rest % 128 | 128 : rest)
                if (rest < 128) break
            }
        }
        at = end
    }

    const bytes = new Uint8Array(HEADER_BYTES + out.length)
    const view = new DataView(bytes.buffer)
    for (let i = 0; i < MAGIC.length; i++) view.setUint8(i, MAGIC.charCodeAt(i))
    view.setUint32(4, VERSION, true)
    view.setUint32(8, kinds.length, true)
    bytes.set(out, HEADER_BYTES)
    return bytes
}

export function decodeTerrain(buffer: ArrayBuffer, points: number): Uint8Array {
    if (buffer.byteLength < HEADER_BYTES) throw new Error(`terrain file is truncated: ${buffer.byteLength} bytes`)
    const view = new DataView(buffer)
    let magic = ""
    for (let i = 0; i < MAGIC.length; i++) magic += String.fromCharCode(view.getUint8(i))
    if (magic !== MAGIC) throw new Error(`not a terrain file: expected magic "${MAGIC}", got "${magic}"`)

    const version = view.getUint32(4, true)
    if (version !== VERSION) throw new Error(`unsupported terrain format version ${version}, expected ${VERSION}`)
    const held = view.getUint32(8, true)
    if (held !== points) throw new Error(`the terrain is of ${held} points, the lattice has ${points}`)

    const bytes = new Uint8Array(buffer)
    const kinds = new Uint8Array(points)
    let filled = 0
    let at = HEADER_BYTES
    while (at < bytes.length) {
        const kind = bytes[at] >> 4
        if (kind >= KINDS) throw new Error(`a run of kind ${kind} at byte ${at}, past the ${KINDS} kinds`)
        let length = (bytes[at++] & LONG) + 1
        if (length > LONG) {
            for (let scale = 1; ; scale *= 128) {
                if (at >= bytes.length) throw new Error("terrain file is truncated in a run's length")
                length += (bytes[at] & 127) * scale
                if (bytes[at++] < 128) break
            }
        }
        if (filled + length > points) throw new Error(`the runs hold more than the lattice's ${points} points`)
        kinds.fill(kind, filled, filled + length)
        filled += length
    }
    if (filled !== points) throw new Error(`the runs hold ${filled} points, the lattice has ${points}`)
    return kinds
}
