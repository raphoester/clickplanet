import {describe, expect, it} from "vitest"
import {BOARDS} from "../app/viewer/boardsAsset.ts"
import {bytesOf} from "../testdata/boards.ts"
import {type BorderLineData, decodeBorderLines} from "./borderLinesFile.ts"
import {buffersOf, type OutlineSet, outlineSegments, outlineSetsOf} from "./outlineSets.ts"
import {CAP_FLOATS, chunkCount} from "./tileChunks.ts"

function lines(runs: number[][], sides: number[] = runs.flatMap(() => [0, 1])): BorderLineData {
    return {runs: Uint32Array.from(runs, (run) => run.length / 3), sides: Uint16Array.from(sides), corners: Int16Array.from(runs.flat())}
}

describe("outlineSegments", () => {
    const at = (points: Int16Array, piece: number) => [...points.slice(piece * 3, piece * 3 + 3)]

    it("draws each cell edge as one piece, from corner to corner", () => {
        const {from, to} = outlineSegments(lines([[1000, 0, 0, 1000, 500, 0, 1000, 900, 0]]))
        expect([...from]).toEqual([1000, 0, 0, 1000, 500, 0])
        expect([...to]).toEqual([1000, 500, 0, 1000, 900, 0])
    })

    it("closes a run that came back to where it started", () => {
        const loop = [0, 0, 30000, 0, 30000, 0, 30000, 0, 0, 0, 0, 30000]
        const {from, to} = outlineSegments(lines([loop]))
        expect(from.length / 3).toBe(3)
        expect(at(to, 2)).toEqual(at(from, 0))
    })

    it("never joins the end of one run to the start of the next", () => {
        const one = [0, 0, 30000, 0, 10000, 30000]
        const other = [30000, 0, 0, 30000, 10000, 0]
        const {from, to} = outlineSegments(lines([one, other]))

        expect(from.length / 3).toBe(2)
        expect(at(to, 0)).toEqual([0, 10000, 30000])
        expect(at(from, 1)).toEqual([30000, 0, 0])
    })
})

// Every piece in exactly one chunk, in order, and each cap holding both ends of its pieces.
function expectChunked({segments: {from, to}, chunks}: OutlineSet) {
    const count = chunkCount(chunks)
    expect(chunks.starts[0]).toBe(0)
    expect(chunks.starts[count]).toBe(from.length / 3)

    let outside = 0
    for (let chunk = 0; chunk < count; chunk++) {
        const [x, y, z, radius] = chunks.caps.subarray(chunk * CAP_FLOATS, chunk * CAP_FLOATS + 4)
        for (let piece = chunks.starts[chunk]; piece < chunks.starts[chunk + 1]; piece++) {
            for (const ends of [from, to]) {
                const px = ends[piece * 3], py = ends[piece * 3 + 1], pz = ends[piece * 3 + 2]
                const across = Math.hypot(py * z - pz * y, pz * x - px * z, px * y - py * x)
                if (Math.atan2(across, px * x + py * y + pz * z) > radius) outside++
            }
        }
    }
    expect(outside).toBe(0)
}

describe.each(Object.values(BOARDS))("the outline sets of $name", (board) => {
    const data = decodeBorderLines(bytesOf(board, board.page.borderLines))
    const tolerances = [0.5, 2].map((edges) => edges * board.spacing / Math.sqrt(3))
    const sets = outlineSetsOf(data, tolerances)

    it("are the cell edges, and a coarse set for each tolerance", () => {
        expect(sets.cells.lines).toBe(data)
        expect(sets.cells.segments).toEqual(outlineSegments(data))
        expect(sets.coarse.map(({tolerance}) => tolerance)).toEqual(tolerances)
        for (const {lines, segments, standsFor} of sets.coarse) {
            expect(segments).toEqual(outlineSegments(lines))
            expect(standsFor.length).toBe(segments.from.length / 3 * 2)
        }
    })

    it("cut each set into chunks that hold every piece once, in order, and both ends of each", () => {
        for (const set of [sets.cells, ...sets.coarse]) expectChunked(set)
    })

    it("name each buffer they hold once, to be handed over by a worker", () => {
        const buffers = buffersOf(sets)
        expect(new Set(buffers).size).toBe(buffers.length)
        const arrays = [sets.cells, ...sets.coarse].flatMap(({lines, segments, chunks}) => [
            lines.runs, lines.sides, lines.corners, segments.from, segments.to, chunks.starts, chunks.caps, chunks.blocks,
        ])
        for (const array of [...arrays, ...sets.coarse.map(({standsFor}) => standsFor)]) {
            expect(buffers).toContain(array.buffer)
        }
    })
})
