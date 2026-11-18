// The countries' outlines of a board, as `npm run borderLines` writes them and the globe reads them (see the page's
// docs/country-outlines.md and /map/README.md#format).

export type BorderLineData = {
    // Each run's count of corners.
    runs: Uint32Array
    // The two landmasses either side of each run, the sea 0.
    sides: Uint16Array
    // Every run's corners in turn, x y z each × 32767.
    corners: Int16Array
}

const MAGIC = "CPBL"
const VERSION = 3
const HEAD_BYTES = 16

// Version 3: each run's first corner as it is, and each next one as its difference from the one before, three zigzag
// varints. Two corners side by side are one cell edge apart, a few dozen units, so most differences take a byte: half
// version 2's size, and a third of it packed.
export function encodeBorderLines({runs, sides, corners}: BorderLineData): Uint8Array {
    const cornerCount = runs.reduce((n, count) => n + count, 0)
    if (sides.length !== runs.length * 2) throw new Error(`${runs.length} runs but ${sides.length / 2} pairs of landmasses`)
    if (corners.length !== cornerCount * 3) throw new Error(`the runs hold ${cornerCount} corners but ${corners.length / 3} are given`)
    if (runs.some((count) => count < 1)) throw new Error("a run holds no corner")

    const steps: number[] = []
    const firsts = new Int16Array(runs.length * 3)
    let at = 0
    runs.forEach((count, run) => {
        firsts.set(corners.subarray(at * 3, at * 3 + 3), run * 3)
        for (let corner = at + 1; corner < at + count; corner++) {
            for (let axis = 0; axis < 3; axis++) pushVarint(steps, zigzag(corners[corner * 3 + axis] - corners[(corner - 1) * 3 + axis]))
        }
        at += count
    })

    const bytes = new Uint8Array(HEAD_BYTES + runs.byteLength + sides.byteLength + firsts.byteLength + steps.length)
    const head = new DataView(bytes.buffer)
    for (let i = 0; i < MAGIC.length; i++) head.setUint8(i, MAGIC.charCodeAt(i))
    head.setUint32(4, VERSION, true)
    head.setUint32(8, runs.length, true)
    head.setUint32(12, cornerCount, true)
    let offset = HEAD_BYTES
    for (const block of [runs, sides, firsts]) {
        bytes.set(new Uint8Array(block.buffer, block.byteOffset, block.byteLength), offset)
        offset += block.byteLength
    }
    bytes.set(steps, offset)
    return bytes
}

export function decodeBorderLines(buffer: ArrayBuffer): BorderLineData {
    if (buffer.byteLength < HEAD_BYTES) throw new Error(`border-line blob is truncated: ${buffer.byteLength} bytes`)
    const head = new DataView(buffer)
    const magic = String.fromCharCode(head.getUint8(0), head.getUint8(1), head.getUint8(2), head.getUint8(3))
    if (magic !== MAGIC) throw new Error(`not a border-line blob: magic "${magic}"`)
    const version = head.getUint32(4, true)
    if (version !== VERSION) throw new Error(`border-line blob version ${version}, expected ${VERSION}`)

    const runCount = head.getUint32(8, true)
    const cornerCount = head.getUint32(12, true)
    const stepsAt = HEAD_BYTES + runCount * 14
    if (buffer.byteLength < stepsAt) throw new Error(`border-line blob is truncated: ${runCount} runs need ${stepsAt} bytes`)
    const runs = new Uint32Array(buffer, HEAD_BYTES, runCount)
    const sides = new Uint16Array(buffer, HEAD_BYTES + runCount * 4, runCount * 2)
    const firsts = new Int16Array(buffer, HEAD_BYTES + runCount * 8, runCount * 3)
    const steps = new Uint8Array(buffer, stepsAt)

    const corners = new Int16Array(cornerCount * 3)
    let at = 0
    let read = 0
    for (let run = 0; run < runCount; run++) {
        const count = runs[run]
        if (count < 1 || at + count > cornerCount) throw new Error(`run ${run} holds ${count} corners, past the blob's ${cornerCount}`)
        corners[at * 3] = firsts[run * 3]
        corners[at * 3 + 1] = firsts[run * 3 + 1]
        corners[at * 3 + 2] = firsts[run * 3 + 2]
        for (let corner = (at + 1) * 3; corner < (at + count) * 3; corner++) {
            let value = 0
            let shift = 0
            let byte: number
            do {
                if (read === steps.length) throw new Error("border-line blob is truncated in its corners")
                byte = steps[read++]
                value |= (byte & 0x7f) << shift
                shift += 7
            } while (byte & 0x80)
            corners[corner] = corners[corner - 3] + ((value >>> 1) ^ -(value & 1))
        }
        at += count
    }
    if (at !== cornerCount) throw new Error(`the runs hold ${at} corners, the blob says ${cornerCount}`)
    if (read !== steps.length) throw new Error(`${steps.length - read} bytes past the last corner`)
    return {runs, sides, corners}
}

function zigzag(value: number): number {
    return ((value << 1) ^ (value >> 31)) >>> 0
}

function pushVarint(out: number[], value: number): void {
    while (value >= 0x80) {
        out.push((value & 0x7f) | 0x80)
        value >>>= 7
    }
    out.push(value)
}
