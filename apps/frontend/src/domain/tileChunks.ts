import {joinSpans, type Span} from "./spans.ts"

export type TileChunks = {
    // Chunk k holds the 0-based tiles starts[k] to starts[k + 1] - 1: one entry more than the chunks.
    starts: Uint32Array
    // CAP_FLOATS per chunk: its cap's unit centre, radius in radians, the radius's cosine and sine, the centre-to-edge chord.
    caps: Float32Array
    // A cap per CHUNKS_PER_BLOCK consecutive chunks.
    blocks: Float32Array
}

export const CAP_FLOATS = 7
export const CHUNKS_PER_BLOCK = 32

// Radians from a chunk's first tile, about 640 km.
export const CHUNK_REACH = 0.1
export const CHUNK_MOST = 1024

// Float32 positions and centres: a cap a little wider than its tiles never leaves one out.
const CAP_SLACK = 1e-4

// per: the points of each item, one after the other; a chunk's cap holds every point of its items.
export function chunksOf(positions: ArrayLike<number>, reach = CHUNK_REACH, most = CHUNK_MOST, per = 1): TileChunks {
    const items = positions.length / 3 / per
    const starts: number[] = []
    const least = Math.cos(reach)
    let anchor = -1
    for (let item = 0; item < items; item++) {
        if (anchor >= 0 && item - anchor < most && cosineOf(positions, item * per * 3, positions, anchor * per * 3) >= least) continue
        starts.push(item)
        anchor = item
    }
    starts.push(items)

    const count = starts.length - 1
    const caps = new Float32Array(count * CAP_FLOATS)
    for (let chunk = 0; chunk < count; chunk++) {
        const centre = directionOf(positions, 3, starts[chunk] * per, starts[chunk + 1] * per)
        let nearest = 1
        for (let point = starts[chunk] * per; point < starts[chunk + 1] * per; point++) {
            nearest = Math.min(nearest, cosineOf(positions, point * 3, centre, 0))
        }
        writeCap(caps, chunk, centre, Math.acos(nearest))
    }

    const blocks = new Float32Array(Math.ceil(count / CHUNKS_PER_BLOCK) * CAP_FLOATS)
    for (let block = 0; block * CHUNKS_PER_BLOCK < count; block++) {
        const from = block * CHUNKS_PER_BLOCK, to = Math.min(count, from + CHUNKS_PER_BLOCK)
        const centre = directionOf(caps, CAP_FLOATS, from, to)
        let radius = 0
        for (let chunk = from; chunk < to; chunk++) {
            const at = chunk * CAP_FLOATS
            radius = Math.max(radius, Math.acos(cosineOf(caps, at, centre, 0)) + caps[at + 3])
        }
        writeCap(blocks, block, centre, radius)
    }

    return {starts: Uint32Array.from(starts), caps, blocks}
}

function directionOf(points: ArrayLike<number>, stride: number, from: number, to: number): Float64Array {
    let x = 0, y = 0, z = 0
    for (let point = from; point < to; point++) {
        x += points[point * stride]
        y += points[point * stride + 1]
        z += points[point * stride + 2]
    }
    const length = Math.sqrt(x * x + y * y + z * z)
    return Float64Array.of(x / length, y / length, z / length)
}

function writeCap(caps: Float32Array, at: number, [x, y, z]: Float64Array, angle: number) {
    const radius = Math.min(Math.PI, angle + CAP_SLACK)
    caps.set([x, y, z, radius, Math.cos(radius), Math.sin(radius), 2 * Math.sin(radius / 2)], at * CAP_FLOATS)
}

export function chunkCount(chunks: TileChunks): number {
    return chunks.starts.length - 1
}

// The chunk holding a tile, by its 0-based index.
export function chunkOf(chunks: TileChunks, tile: number): number {
    let low = 0, high = chunks.starts.length - 2
    while (low < high) {
        const middle = (low + high + 1) >> 1
        if (chunks.starts[middle] <= tile) low = middle
        else high = middle - 1
    }
    return low
}

export type ChunkView = {
    // three's column-major projection × view of an orthographic camera, so w stays 1.
    toClip: ArrayLike<number>
    // Unit, from the earth's centre toward the camera.
    back: readonly [number, number, number]
    // Must match the display vertex shader, which drops a tile at unit ground g while g · back < -limb.
    limb: number
    // In world units: how far a sprite or a blast's throw reaches past its ground.
    reach: number
}

const OUT = 0
const PART = 1
const IN = 2

// Writes the chunks that may show into `shown`, in order, and answers how many: `shown` holds one entry per chunk.
export function showingChunks(chunks: TileChunks, {toClip: m, back, limb, reach}: ChunkView, shown: Uint32Array, among?: (chunk: number) => boolean): number {
    const [bx, by, bz] = back
    const beyond = Math.acos(Math.min(1, Math.max(-1, -limb)))
    const cosBeyond = Math.cos(beyond), sinBeyond = Math.sin(beyond)
    const sx = Math.sqrt(m[0] * m[0] + m[4] * m[4] + m[8] * m[8])
    const sy = Math.sqrt(m[1] * m[1] + m[5] * m[5] + m[9] * m[9])
    const m0 = m[0], m4 = m[4], m8 = m[8], m12 = m[12], m1 = m[1], m5 = m[5], m9 = m[9], m13 = m[13]

    // OUT when no tile of the cap can show, IN when every one is in front of the earth and on screen.
    const sightOf = (caps: Float32Array, at: number) => {
        const x = caps[at], y = caps[at + 1], z = caps[at + 2]
        const radius = caps[at + 3], cos = caps[at + 4], sin = caps[at + 5]
        const facing = x * bx + y * by + z * bz
        if (radius + beyond < Math.PI && facing < cos * cosBeyond - sin * sinBeyond) return OUT

        const spread = caps[at + 6] + reach
        const cx = Math.abs(m0 * x + m4 * y + m8 * z + m12)
        const cy = Math.abs(m1 * x + m5 * y + m9 * z + m13)
        if (cx > 1 + sx * spread || cy > 1 + sy * spread) return OUT
        if (cx + sx * spread > 1 || cy + sy * spread > 1 || facing <= -cos) return PART

        return facing * cos - Math.sqrt(1 - facing * facing) * sin >= -limb ? IN : PART
    }

    const count = chunkCount(chunks)
    let written = 0
    for (let block = 0, from = 0; from < count; block++, from += CHUNKS_PER_BLOCK) {
        const seen = sightOf(chunks.blocks, block * CAP_FLOATS)
        if (seen === OUT) continue
        const to = Math.min(count, from + CHUNKS_PER_BLOCK)
        for (let chunk = from; chunk < to; chunk++) {
            if (among && !among(chunk)) continue
            if (seen === IN || sightOf(chunks.caps, chunk * CAP_FLOATS) !== OUT) shown[written++] = chunk
        }
    }
    return written
}

export function drawnRanges(chunks: TileChunks, shown: Uint32Array, count: number, most: number, least = 0): Span[] {
    const spans: Span[] = []
    let last: Span | undefined
    for (let i = 0; i < count; i++) {
        const chunk = shown[i]
        const start = chunks.starts[chunk], end = chunks.starts[chunk + 1]
        if (last && start - (last.start + last.count) < Math.max(least, 1)) {
            last.count = end - last.start
        } else {
            last = {start, count: end - start}
            spans.push(last)
        }
    }
    return joinSpans(spans, most, least)
}

function cosineOf(a: ArrayLike<number>, aAt: number, b: ArrayLike<number>, bAt: number): number {
    const ax = a[aAt], ay = a[aAt + 1], az = a[aAt + 2]
    const bx = b[bAt], by = b[bAt + 1], bz = b[bAt + 2]
    const cosine = (ax * bx + ay * by + az * bz) / Math.sqrt((ax * ax + ay * ay + az * az) * (bx * bx + by * by + bz * bz))
    return Math.max(-1, Math.min(1, cosine))
}
