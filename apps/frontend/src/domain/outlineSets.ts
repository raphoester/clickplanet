import type {BorderLineData} from "./borderLinesFile.ts"
import {coarseOutlinesOf} from "./coarseOutline.ts"
import {CHUNK_MOST, CHUNK_REACH, chunksOf, type TileChunks} from "./tileChunks.ts"

// A piece runs from one corner of a run to the next: a run of n corners is n - 1 pieces.
export type Segments = {from: Int16Array, to: Int16Array}

export function outlineSegments(data: BorderLineData): Segments {
    const pieces = data.corners.length / 3 - data.runs.length
    const from = new Int16Array(pieces * 3)
    const to = new Int16Array(pieces * 3)

    let at = 0, piece = 0
    for (const corners of data.runs) {
        from.set(data.corners.subarray(at * 3, (at + corners - 1) * 3), piece * 3)
        to.set(data.corners.subarray((at + 1) * 3, (at + corners) * 3), piece * 3)
        piece += corners - 1
        at += corners
    }
    return {from, to}
}

// A cap holds both ends of each of its pieces.
export function outlineChunksOf({from, to}: Segments): TileChunks {
    const pieces = from.length / 3
    const ends = new Int16Array(pieces * 6)
    for (let piece = 0; piece < pieces; piece++) {
        ends.set(from.subarray(piece * 3, piece * 3 + 3), piece * 6)
        ends.set(to.subarray(piece * 3, piece * 3 + 3), piece * 6 + 3)
    }
    return chunksOf(ends, CHUNK_REACH, CHUNK_MOST, 2)
}

// One set of the outline's pieces, culled on its own.
export type OutlineSet = {lines: BorderLineData, segments: Segments, chunks: TileChunks}

// tolerance: how far its dropped corners lie from the cell edges, at most; standsFor: see CoarseOutline.
export type CoarseSet = OutlineSet & {tolerance: number, standsFor: Float32Array}

// The cell edges, and a coarse set for each tolerance.
export type OutlineSets = {cells: OutlineSet, coarse: CoarseSet[]}

export function outlineSetsOf(data: BorderLineData, tolerances: readonly number[]): OutlineSets {
    const setOf = (lines: BorderLineData): OutlineSet => {
        const segments = outlineSegments(lines)
        return {lines, segments, chunks: outlineChunksOf(segments)}
    }
    return {
        cells: setOf(data),
        coarse: coarseOutlinesOf(data, tolerances).map(({lines, standsFor}, at) => ({...setOf(lines), tolerance: tolerances[at], standsFor})),
    }
}

// Every buffer the sets hold, once: a worker hands them over rather than copies them.
export function buffersOf({cells, coarse}: OutlineSets): ArrayBuffer[] {
    const buffers = new Set<ArrayBufferLike>()
    for (const {lines, segments, chunks} of [cells, ...coarse]) {
        for (const array of [lines.runs, lines.sides, lines.corners, segments.from, segments.to, chunks.starts, chunks.caps, chunks.blocks]) {
            buffers.add(array.buffer)
        }
    }
    for (const {standsFor} of coarse) buffers.add(standsFor.buffer)
    return [...buffers] as ArrayBuffer[]
}
