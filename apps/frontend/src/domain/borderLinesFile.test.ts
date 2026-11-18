import {describe, expect, it} from "vitest"
import {type BorderLineData, decodeBorderLines, encodeBorderLines} from "./borderLinesFile.ts"
import {BOARDS} from "../app/viewer/boardsAsset.ts"
import {type Lattice, unwarp} from "./lattice.ts"
import {type LatticeFace, latticeFaces} from "./latticeCells.ts"
import {decodeProvinceLines} from "./provinceLinesFile.ts"
import {bytesOf} from "../testdata/boards.ts"

function lines(runs: number[][], sides: number[] = runs.flatMap(() => [0, 1])): BorderLineData {
    return {runs: Uint32Array.from(runs, (run) => run.length / 3), sides: Uint16Array.from(sides), corners: Int16Array.from(runs.flat())}
}

function bufferOf(bytes: Uint8Array): ArrayBuffer {
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

function plain({runs, sides, corners}: BorderLineData) {
    return {runs: [...runs], sides: [...sides], corners: [...corners]}
}

// How far a direction is from the nearest place three of the ground's cells meet, or two meet on an edge of the
// icosahedron, in grid steps: the ground finds a cell on its face's grid (latticeCells.ts), so they meet at a triangle's
// centroid, a third of a step in from each of its corners, and a cell edge bends from one face into the next at the
// middle of its lattice edge.
function offCellCorner(lattice: Lattice, faces: readonly LatticeFace[], x: number, y: number, z: number): number {
    const length = Math.hypot(x, y, z)
    const ground = [x / length, y / length, z / length]
    const dot = (a: readonly number[], b: readonly number[]) => a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
    let best = 0
    faces.forEach(({normal}, face) => {
        if (dot(ground, normal) > dot(ground, faces[best].normal)) best = face
    })
    const {normal, height, origin, alongOf, downOf} = faces[best]
    const reach = height / dot(ground, normal)
    const onPlane = [0, 1, 2].map((k) => ground[k] * reach - origin[k])
    const weights = [dot(onPlane, alongOf), dot(onPlane, downOf)]
    if (lattice.warp !== null) {
        const back = new Float64Array(2)
        unwarp(lattice.warp, weights[0], weights[1], back)
        weights.splice(0, 2, back[0], back[1])
    }
    const [gx, gy] = weights.map((w) => w * lattice.cols)
    const fx = Math.floor(gx), fy = Math.floor(gy)
    const places = [[fx + 1 / 3, fy + 1 / 3], [fx + 2 / 3, fy + 2 / 3], [fx + 0.5, 0], [0, fy + 0.5], [fx + 0.5, lattice.cols - fx - 0.5]]
    // On the grid of equilateral triangles: (x, y) is x + y/2 across and y·√3/2 up.
    return Math.min(...places.map(([px, py]) => Math.hypot(gx - px + (gy - py) / 2, (gy - py) * Math.sqrt(3) / 2)))
}

describe("the border lines' file", () => {
    it("reads back the runs, the landmasses either side of each and the corners", () => {
        const written = lines([[1, 2, 3, 4, 5, 6], [7, 8, 9, 10, 11, 12, 13, 14, 15]], [0, 4, 4, 9])
        expect(plain(decodeBorderLines(bufferOf(encodeBorderLines(written))))).toEqual(plain(written))
    })

    it("reads back corners as far apart as the int16s go, and a run of one corner", () => {
        const written = lines([[-32767, 32767, 0, 32767, -32767, -1, 0, 0, 0, 5, -5, 300], [12, -12, 7]])
        expect(plain(decodeBorderLines(bufferOf(encodeBorderLines(written))))).toEqual(plain(written))
    })

    it("takes a byte a step between two corners a cell edge apart", () => {
        const step = encodeBorderLines(lines([[1000, 2000, 3000, 1040, 1950, 3010]])).length
        const still = encodeBorderLines(lines([[1000, 2000, 3000]])).length
        expect(step - still).toBe(3)
    })

    it("refuses a blob that is not one", () => {
        const bytes = encodeBorderLines(lines([[1, 2, 3, 4, 5, 6]]))
        bytes[0] = "X".charCodeAt(0)
        expect(() => decodeBorderLines(bufferOf(bytes))).toThrow(/magic/)
    })

    it("refuses version 2, whose corners were plain int16s", () => {
        const bytes = encodeBorderLines(lines([[1, 2, 3, 4, 5, 6]]))
        new DataView(bytes.buffer).setUint32(4, 2, true)
        expect(() => decodeBorderLines(bufferOf(bytes))).toThrow(/version 2/)
    })

    it("refuses a blob cut short, and one with bytes past its last corner", () => {
        const bytes = encodeBorderLines(lines([[1, 2, 3, 400, 500, 600]]))
        expect(() => decodeBorderLines(bufferOf(bytes.slice(0, bytes.length - 1)))).toThrow(/truncated/)
        expect(() => decodeBorderLines(bufferOf(Uint8Array.from([...bytes, 0])))).toThrow(/past the last corner/)
    })

    for (const board of Object.values(BOARDS)) {
        it(`writes ${board.name}'s outline back to the same bytes`, () => {
            const shipped = bytesOf(board, board.page.borderLines)
            expect(Buffer.from(encodeBorderLines(decodeBorderLines(shipped))).equals(Buffer.from(shipped))).toBe(true)
        })

        // Until 2026-11-18 a corner was the circumcentre of its lattice triangle, 4% to 7.5% of a spacing off the corner
        // the ground draws on earth-2: the line ran beside the hexagons, plainly on a coast.
        it(`draws ${board.name}'s lines on the ground's cells, every corner where they meet to within the int16s`, () => {
            const files = [decodeBorderLines(bytesOf(board, board.page.borderLines))]
            if (board.page.provinceLines) files.push(decodeProvinceLines(bytesOf(board, board.page.provinceLines)).lines)
            const faces = latticeFaces(board.lattice)
            for (const {corners} of files) {
                let worst = 0
                for (let at = 0; at < corners.length; at += 3) {
                    worst = Math.max(worst, offCellCorner(board.lattice, faces, corners[at] / 32767, corners[at + 1] / 32767, corners[at + 2] / 32767))
                }
                // An int16 step is 3.1e-5 of the radius, at most 0.015 of a grid step on earth-2 (cols 590).
                expect(worst).toBeLessThan(0.02)
            }
        })
    }
})
