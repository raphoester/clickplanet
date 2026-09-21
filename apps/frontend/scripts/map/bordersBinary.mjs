// The borders blob's layout, little-endian:
//
//   uint32 header length | JSON {"tiles": N, "codes": [...]} | N uint16 landmass | 0-2 pad
//   | codes.length * 5 f32 frames | codes.length uint32 totals
//
// Landmass 0 is no country; `codes[k]` is landmass k's ISO code, so one country has many landmasses.
//
// **Both the header and the landmass table are padded to 4 bytes**, so the float section stays
// aligned and the browser's decoder can take zero-copy views — `loadBorders` in
// src/app/viewer/borderField.ts is the other end. The second pad is not decoration: an odd tile
// count leaves the table on a 2-byte boundary, and `new Float32Array(buffer, at, …)` then throws
// "start offset should be a multiple of 4" and the globe does not load at all. Every map until now
// happened to have an even tile count, so this only appeared when one did not.
import {createHash} from "node:crypto"

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
