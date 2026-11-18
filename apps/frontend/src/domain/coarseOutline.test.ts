import {describe, expect, it} from "vitest"
import {BOARDS} from "../app/viewer/boardsAsset.ts"
import {bytesOf} from "../testdata/boards.ts"
import {type BorderLineData, decodeBorderLines} from "./borderLinesFile.ts"
import {type CoarseOutline, coarseOutlinesOf, directionsOf, LEAST_SLANT, MOST_DIP, offsetOf} from "./coarseOutline.ts"

function lines(runs: number[][], sides: number[] = runs.flatMap(() => [0, 1])): BorderLineData {
    return {runs: Uint32Array.from(runs, (run) => run.length / 3), sides: Uint16Array.from(sides), corners: Int16Array.from(runs.flat())}
}

// Corners near the pole, x and y in int16 units on the tangent plane.
function near(points: number[][]): number[] {
    return points.flatMap(([x, y]) => [x, y, 32000])
}

// Cell edges 30 units long, 30° across the run: its corners are 15 units either side of a line.
const ZIGZAG = near(Array.from({length: 41}, (_, at) => [at * 26, at % 2 ? 15 : 0]))

// What each coarse piece stands for: the cell edges' corners from its first to its last, by their index in the lines.
type Kept = {first: number, last: number, run: number}

function keptOf(data: BorderLineData, {lines: coarse}: CoarseOutline): Kept[] {
    const same = (a: number, b: number) => [0, 1, 2].every((axis) => data.corners[a * 3 + axis] === coarse.corners[b * 3 + axis])
    const pieces: Kept[] = []
    let at = 0, coarseAt = 0
    data.runs.forEach((count, run) => {
        let previous = at
        expect(same(at, coarseAt), `run ${run} keeps its first corner`).toBe(true)
        for (let corner = 1; corner < coarse.runs[run]; corner++) {
            let found = previous + 1
            while (!same(found, coarseAt + corner)) found++
            expect(found).toBeLessThan(at + count)
            pieces.push({first: previous, last: found, run})
            previous = found
        }
        expect(previous, `run ${run} keeps its last corner`).toBe(at + count - 1)
        at += count
        coarseAt += coarse.runs[run]
    })
    return pieces
}

function farthestDropped(data: BorderLineData, outline: CoarseOutline): number {
    const directions = directionsOf(data.corners)
    let farthest = 0
    for (const {first, last} of keptOf(data, outline)) {
        for (let corner = first + 1; corner < last; corner++) farthest = Math.max(farthest, offsetOf(directions, corner, first, last))
    }
    return farthest
}

describe("a coarse outline", () => {
    it("keeps each run's ends and landmasses, and drops the corners that stay within the tolerance", () => {
        const data = lines([ZIGZAG, near([[0, 3000], [200, 3100], [400, 3000]])], [0, 4, 4, 9])
        const [outline] = coarseOutlinesOf(data, [20 / 32000])

        expect([...outline.lines.sides]).toEqual([0, 4, 4, 9])
        expect(outline.lines.runs[0]).toBeLessThan(ZIGZAG.length / 3 / 4)
        expect(farthestDropped(data, outline)).toBeLessThanOrEqual(20 / 32000)
    })

    it("keeps every corner when the tolerance is less than the zigzag of the cell edges", () => {
        const data = lines([ZIGZAG])
        const [outline] = coarseOutlinesOf(data, [10 / 32000])
        expect([...outline.lines.corners]).toEqual([...data.corners])
    })

    it("says how many cell edges each piece stands for, and how much shorter its chord is than they are", () => {
        const data = lines([ZIGZAG])
        const [outline] = coarseOutlinesOf(data, [20 / 32000])
        const pieces = keptOf(data, outline)
        const d = directionsOf(data.corners)
        const distance = (a: number, b: number) => Math.hypot(d[b * 3] - d[a * 3], d[b * 3 + 1] - d[a * 3 + 1], d[b * 3 + 2] - d[a * 3 + 2])

        expect(pieces.length).toBeGreaterThan(1)
        expect(outline.standsFor.length).toBe(pieces.length * 2)
        pieces.forEach(({first, last}, piece) => {
            let length = 0
            for (let corner = first; corner < last; corner++) length += distance(corner, corner + 1)
            expect(outline.standsFor[piece * 2]).toBe(last - first)
            expect(outline.standsFor[piece * 2 + 1]).toBeCloseTo(distance(first, last) / length, 5)
            expect(outline.standsFor[piece * 2 + 1]).toBeCloseTo(Math.cos(Math.PI / 6), 1)
        })
    })

    it("keeps a ring's corner farthest from where it starts, so a small lake is never a point", () => {
        const ring = near([[0, 0], [100, -60], [200, 0], [200, 120], [100, 180], [0, 120], [0, 0]])
        const [outline] = coarseOutlinesOf(lines([ring]), [1])
        const kept = Array.from({length: outline.lines.runs[0]}, (_, at) => [...outline.lines.corners.subarray(at * 3, at * 3 + 2)])
        expect(kept).toEqual([[0, 0], [200, 120], [0, 0]])
    })

    it("lets a run of one corner be", () => {
        const [outline] = coarseOutlinesOf(lines([[1, 2, 30000]]), [1])
        expect([...outline.lines.runs]).toEqual([1])
        expect(outline.standsFor.length).toBe(0)
    })
})

describe.each(Object.values(BOARDS))("the coarse outlines of $name", (board) => {
    const data = decodeBorderLines(bytesOf(board, board.page.borderLines))
    const cellEdge = board.spacing / Math.sqrt(3)
    const tolerances = [0.5, 1, 2].map((edges) => edges * cellEdge)
    const outlines = coarseOutlinesOf(data, tolerances)

    it("drop no corner farther than the tolerance from its piece, and keep every run's ends", () => {
        outlines.forEach((outline, at) => {
            expect(farthestDropped(data, outline), `tolerance ${tolerances[at]}`).toBeLessThanOrEqual(tolerances[at])
            expect(outline.lines.runs.length).toBe(data.runs.length)
            expect(outline.lines.sides).toBe(data.sides)
        })
    })

    it("hold each run's cell edges, a slant between the least and 1 for each piece", () => {
        for (const outline of outlines) {
            const edges = outline.standsFor.filter((_, at) => at % 2 === 0).reduce((sum, count) => sum + count, 0)
            expect(edges).toBe(data.corners.length / 3 - data.runs.length)
            const slants = outline.standsFor.filter((_, at) => at % 2 === 1)
            expect(Math.min(...slants)).toBeGreaterThanOrEqual(LEAST_SLANT)
            expect(Math.max(...slants)).toBeLessThanOrEqual(1 + 1e-6)
        }
    })

    it("never dip a piece's middle under its ends by more than the most", () => {
        for (const {lines: {corners, runs}} of outlines) {
            const d = directionsOf(corners)
            let at = 0, deepest = 0
            for (const count of runs) {
                for (let corner = at; corner < at + count - 1; corner++) {
                    const a = corner * 3, b = a + 3
                    deepest = Math.max(deepest, 1 - Math.hypot(d[a] + d[b], d[a + 1] + d[b + 1], d[a + 2] + d[b + 2]) / 2)
                }
                at += count
            }
            expect(deepest).toBeGreaterThan(0)
            expect(deepest).toBeLessThanOrEqual(MOST_DIP)
        }
    })

    it("draw fewer pieces the coarser they are", () => {
        const counts = outlines.map(({lines: {corners, runs}}) => corners.length / 3 - runs.length)
        expect(counts).toEqual([...counts].sort((a, b) => b - a))
        expect(counts[0]).toBeLessThan(data.corners.length / 3 - data.runs.length)
    })
})

describe("the coarse outlines of earth-2", () => {
    it("draw a fifth of the cell edges' pieces at a cell edge's tolerance", () => {
        const board = BOARDS["earth-2"]
        const data = decodeBorderLines(bytesOf(board, board.page.borderLines))
        const [outline] = coarseOutlinesOf(data, [board.spacing / Math.sqrt(3)])
        const pieces = (lines: BorderLineData) => lines.corners.length / 3 - lines.runs.length
        expect(pieces(outline.lines) / pieces(data)).toBeLessThan(0.25)
    })
})
