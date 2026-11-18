import {describe, expect, it} from "vitest"
import {
    canonicalId,
    CORNERS,
    faceIds,
    type Lattice,
    latticeIds,
    latticeNeighbours,
    latticePositions,
    latticeSize,
    latticeTriangles,
    positionsHash,
    positionWriter,
    threeLattice,
    unwarp,
    walkCanonical,
    warpedWeights,
} from "./lattice.ts"
import {BOARDS} from "../app/viewer/boardsAsset.ts"

// earth-2's warp and rotation, at a few cols.
const warpedAt = (cols: number): Lattice => ({...BOARDS["earth-2"].lattice, cols})

describe("latticePositions", () => {
    it("holds the geodesic sphere's 10*cols^2 + 2 points, each on the unit sphere", () => {
        const positions = latticePositions(5)
        expect(positions.length).toBe(latticeSize(5) * 3)
        for (let point = 0; point < latticeSize(5); point++) {
            const [x, y, z] = positions.subarray(point * 3, point * 3 + 3)
            expect(Math.sqrt(x * x + y * y + z * z)).toBeCloseTo(1, 6)
        }
    })

    it("holds each point once", () => {
        const positions = latticePositions(7)
        const keys = new Set<string>()
        for (let point = 0; point < latticeSize(7); point++) keys.add(positions.subarray(point * 3, point * 3 + 3).join(","))
        expect(keys.size).toBe(latticeSize(7))
    })
})

describe("latticeIds", () => {
    it("gives each point its own canonical id, the twelve corners the first twelve", () => {
        const ids = latticeIds(7)
        const {at} = latticeNeighbours(7)
        expect(new Set(ids).size).toBe(latticeSize(7))
        expect(Math.max(...ids)).toBe(latticeSize(7) - 1)
        const corners = Array.from(ids).filter((_, point) => at[point + 1] - at[point] === 5)
        expect(corners.sort((a, b) => a - b)).toEqual(Array.from({length: 12}, (_, id) => id))
    })
})

describe("latticeNeighbours", () => {
    it("gives every point 6 neighbours and the 12 corners 5, each listed from both ends", () => {
        const cols = 6
        const {at, to} = latticeNeighbours(cols)
        let corners = 0
        for (let point = 0; point < latticeSize(cols); point++) {
            const row = to.subarray(at[point], at[point + 1])
            if (row.length === 5) corners++
            else expect(row.length).toBe(6)
            for (const other of row) expect(to.subarray(at[other], at[other + 1])).toContain(point)
        }
        expect(corners).toBe(12)
    })
})

describe("latticeTriangles", () => {
    it("covers the sphere with 20*cols^2 triangles of three neighbours each", () => {
        const cols = 4
        const corners = latticeTriangles(cols)
        const {at, to} = latticeNeighbours(cols)
        const touch = (a: number, b: number) => to.subarray(at[a], at[a + 1]).includes(b)
        expect(corners.length).toBe(20 * cols * cols * 3)
        for (let t = 0; t < corners.length; t += 3) {
            expect(touch(corners[t], corners[t + 1]) && touch(corners[t + 1], corners[t + 2]) && touch(corners[t], corners[t + 2])).toBe(true)
        }
    })
})

describe("positionsHash", () => {
    it("changes when one bit of one coordinate does", () => {
        const positions = latticePositions(3)
        const before = positionsHash(positions)
        new Uint32Array(positions.buffer)[17] ^= 1
        expect(positionsHash(positions)).not.toBe(before)
    })
})

describe("walkCanonical", () => {
    it("visits every point once, in canonical id order, at the (face, i, j) whose canonical id it is", () => {
        for (const cols of [2, 5, 11]) {
            const faces = faceIds(cols)
            let next = 0
            walkCanonical(cols, (face, i, j, id) => {
                expect(id).toBe(next++)
                expect(canonicalId(cols, faces[face], i, j)).toBe(id)
            })
            expect(next).toBe(latticeSize(cols))
        }
    })
})

describe("a warped and rotated lattice", () => {
    it("is in canonical order: a point is its own canonical id", () => {
        expect(Array.from(latticeIds(warpedAt(7)))).toEqual(Array.from({length: latticeSize(7)}, (_, id) => id))
    })

    it("puts a point on an edge or a corner at the same bits from every face it is on", () => {
        for (const cols of [5, 12]) {
            const lattice = warpedAt(cols)
            const positions = latticePositions(lattice)
            const words = new Uint32Array(positions.buffer)
            const place = positionWriter(lattice)
            const faces = faceIds(cols)
            const out = new Float32Array(3)
            const outWords = new Uint32Array(out.buffer)
            let differing = 0
            for (let face = 0; face < 20; face++) {
                for (let i = 0; i <= cols; i++) {
                    for (let j = 0; j <= cols - i; j++) {
                        const id = canonicalId(cols, faces[face], i, j)
                        place(face, i, j, out, 0)
                        for (let k = 0; k < 3; k++) if (outWords[k] !== words[id * 3 + k]) differing++
                    }
                }
            }
            expect(differing, `cols ${cols}`).toBe(0)
        }
    })

    it("holds each point once, on the unit sphere, its twelve corners at the rotated icosahedron's", () => {
        const lattice = warpedAt(9)
        const positions = latticePositions(lattice)
        const keys = new Set<string>()
        for (let point = 0; point < latticeSize(9); point++) {
            const [x, y, z] = positions.subarray(point * 3, point * 3 + 3)
            expect(Math.sqrt(x * x + y * y + z * z)).toBeCloseTo(1, 6)
            keys.add(`${x},${y},${z}`)
        }
        expect(keys.size).toBe(latticeSize(9))
        const corners = lattice.corners!
        for (let corner = 0; corner < 12; corner++) {
            const [x, y, z] = corners.slice(corner * 3, corner * 3 + 3)
            const length = Math.sqrt(x * x + y * y + z * z)
            expect(Array.from(positions.subarray(corner * 3, corner * 3 + 3))).toEqual([x, y, z].map((value) => Math.fround(value / length)))
        }
    })

    it("turns three's corners by a rotation: their lengths and the angles between them are three's", () => {
        const corners = BOARDS["earth-2"].lattice.corners!
        const dot = (of: readonly number[], a: number, b: number) => of[a * 3] * of[b * 3] + of[a * 3 + 1] * of[b * 3 + 1] + of[a * 3 + 2] * of[b * 3 + 2]
        for (let a = 0; a < 12; a++) {
            for (let b = a; b < 12; b++) expect(dot(corners, a, b)).toBeCloseTo(dot(CORNERS, a, b), 12)
        }
    })

    it("gives every point 6 neighbours and the 12 corners 5, the first twelve canonical ids", () => {
        const {at} = latticeNeighbours(warpedAt(6))
        for (let point = 0; point < latticeSize(6); point++) expect(at[point + 1] - at[point]).toBe(point < 12 ? 5 : 6)
    })

    it("folds no triangle: each turns the way three winds its faces, clockwise from outside", () => {
        for (const lattice of [threeLattice(97), warpedAt(97), warpedAt(400)]) {
            const positions = latticePositions(lattice)
            const corners = latticeTriangles(lattice)
            let folded = 0
            const p = (point: number, k: number) => positions[point * 3 + k]
            for (let t = 0; t < corners.length; t += 3) {
                const [a, b, c] = [corners[t], corners[t + 1], corners[t + 2]]
                const u = [p(b, 0) - p(a, 0), p(b, 1) - p(a, 1), p(b, 2) - p(a, 2)]
                const v = [p(c, 0) - p(a, 0), p(c, 1) - p(a, 1), p(c, 2) - p(a, 2)]
                const normal = [u[1] * v[2] - u[2] * v[1], u[2] * v[0] - u[0] * v[2], u[0] * v[1] - u[1] * v[0]]
                if (normal[0] * p(a, 0) + normal[1] * p(a, 1) + normal[2] * p(a, 2) <= 0) folded++
            }
            expect(folded, `cols ${lattice.cols}`).toBe(0)
        }
    }, 60_000)
})

describe("unwarp", () => {
    it("undoes the warp of every point of earth-2's faces to 1e-7 in two Newton steps, a cell being 1.7e-3", () => {
        const {cols, warp} = BOARDS["earth-2"].lattice
        const weights = new Float64Array(3)
        const back = new Float64Array(2)
        let worst = 0
        for (let i = 0; i <= cols; i++) {
            for (let j = 0; j <= cols - i; j++) {
                warpedWeights(warp, (cols - i - j) / cols, j / cols, i / cols, weights)
                unwarp(warp!, weights[1], weights[2], back)
                worst = Math.max(worst, Math.abs(back[0] - j / cols), Math.abs(back[1] - i / cols))
            }
        }
        expect(worst).toBeLessThan(1e-7)
    })
})

describe("three's lattice", () => {
    it("is what a bare cols names", () => {
        expect(Array.from(latticePositions(7))).toEqual(Array.from(latticePositions(threeLattice(7))))
    })
})
