import {createHash} from "node:crypto"

// Keeps the f32 frames 4-byte aligned, as the browser's Float32Array views require.
const PAD_TO = 4
const padding = (length) => (PAD_TO - (length % PAD_TO)) % PAD_TO

/**
 * @param {{codes: string[], assignment: Uint16Array, frames: Float32Array, totals: Uint32Array}} borders
 * @returns {{bytes: Buffer, fileName: string}}
 */
export function encodeBorders({codes, assignment, frames, totals}) {
    if (frames.length !== codes.length * 5) {
        throw new Error(`expected ${codes.length * 5} frame values, got ${frames.length}`)
    }
    if (totals.length !== codes.length) {
        throw new Error(`expected ${codes.length} totals, got ${totals.length}`)
    }

    let json = JSON.stringify({tiles: assignment.length, codes})
    while (padding(4 + Buffer.byteLength(json, "utf8")) !== 0) json += " "
    const header = Buffer.from(json, "utf8")
    const length = Buffer.alloc(4)
    length.writeUInt32LE(header.length, 0)

    const bytes = Buffer.concat([
        length,
        header,
        Buffer.from(assignment.buffer, assignment.byteOffset, assignment.byteLength),
        Buffer.alloc(padding(assignment.byteLength)),
        Buffer.from(frames.buffer, frames.byteOffset, frames.byteLength),
        Buffer.from(totals.buffer, totals.byteOffset, totals.byteLength),
    ])

    const hash = createHash("sha256").update(bytes).digest("hex").slice(0, 8)
    return {bytes, fileName: `borders-${hash}.bin`}
}
