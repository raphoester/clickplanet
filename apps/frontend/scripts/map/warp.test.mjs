import {describe, expect, it} from "vitest"

import {latticeTriangles} from "../../src/domain/lattice.ts"
import {BOARD_DEFINITIONS, EARTH_2, SIEGE_1, boardDir, readBoard} from "./board.mjs"
import {FIRST_FIT, ROTATION, areaSpread, fitWarp, rotatedCorners} from "./warp.mjs"

const EARTH_KM = 6371

// The twelve corners' places, as docs/boards.md gives them: [lat, lon].
const PLACES = [
    [17.83, -59.98], [68.15, 2.63], [-68.15, -177.37], [-17.83, 120.02], [-42.90, 50.04], [4.71, 3.60],
    [-4.71, -176.40], [42.90, -129.96], [43.48, 137.76], [18.55, 67.14], [-18.55, -112.86], [-43.48, -42.24],
]

describe("rotatedCorners", () => {
    const corners = rotatedCorners()

    it("are three's corners turned by the rotation, at the places docs/boards.md gives", () => {
        expect(corners).toHaveLength(36)
        PLACES.forEach(([lat, lon], corner) => {
            const [x, y, z] = corners.slice(corner * 3, corner * 3 + 3)
            const length = Math.hypot(x, y, z)
            expect(Math.asin(y / length) * 180 / Math.PI, `corner ${corner}`).toBeCloseTo(lat, 2)
            expect(Math.atan2(z, -x) * 180 / Math.PI, `corner ${corner}`).toBeCloseTo(lon, 2)
        })
    })

    it("are a rotation's: the matrix is orthonormal, and turns no corner's length", () => {
        for (let a = 0; a < 3; a++) {
            for (let b = 0; b < 3; b++) {
                const dot = ROTATION[a][0] * ROTATION[b][0] + ROTATION[a][1] * ROTATION[b][1] + ROTATION[a][2] * ROTATION[b][2]
                expect(dot).toBeCloseTo(a === b ? 1 : 0, 12)
            }
        }
        for (let corner = 0; corner < 12; corner++) {
            expect(Math.hypot(...corners.slice(corner * 3, corner * 3 + 3))).toBeCloseTo(Math.sqrt(1 + ((1 + Math.sqrt(5)) / 2) ** 2), 12)
        }
    })

    it("are what each new board's manifest carries", () => {
        for (const board of [EARTH_2, SIEGE_1]) expect(readBoard(boardDir(board.name)).manifest.lattice.corners).toEqual(corners)
    })
})

describe("the twelve five-neighbour points", () => {
    // The nearest land to each, on each new board, in km: at least 181 on earth-2 (the Sea of Japan), 189 on siege-1.
    const LEAST_KM = {"earth-2": 180, "siege-1": 185}

    for (const definition of [EARTH_2, SIEGE_1]) {
        it(`are at sea on ${definition.name}, far from its own coasts`, () => {
            const board = readBoard(boardDir(definition.name))
            expect(Array.from(board.ids).filter((id) => id < 12)).toEqual([])
            const corners = board.manifest.lattice.corners
            for (let corner = 0; corner < 12; corner++) {
                const [x, y, z] = corners.slice(corner * 3, corner * 3 + 3)
                const length = Math.hypot(x, y, z)
                let nearest = -1
                for (let t = 0; t < board.count; t++) {
                    nearest = Math.max(nearest, (board.positions[t * 3] * x + board.positions[t * 3 + 1] * y + board.positions[t * 3 + 2] * z) / length)
                }
                expect(Math.acos(nearest) * EARTH_KM, `corner ${corner}`).toBeGreaterThan(LEAST_KM[definition.name])
            }
        }, 60_000)
    }
})

describe("the warp", () => {
    it("is fitted the same way every time: Nelder-Mead from the first fit, rounded to 8 decimals", () => {
        const once = fitWarp(24)
        expect(fitWarp(24)).toEqual(once)
        expect(once.spread).toBeLessThan(areaSpread({cols: 24, warp: FIRST_FIT, corners: null, order: "canonical"}))
    })

    // The fit is pinned: each board's warp is the least spread of its cells' areas at its own cols, which a nudge of any
    // coefficient only widens. `fitWarp(cols)` writes it again (about 3 minutes at 590).
    for (const {name, lattice} of [EARTH_2, SIEGE_1]) {
        it(`is ${name}'s own, fitted at its cols ${lattice.cols}: the cells' areas spread least there`, () => {
            const shape = {cols: lattice.cols, warp: null, corners: null, order: "canonical"}
            const triangles = latticeTriangles(shape)
            const spread = areaSpread({...shape, warp: lattice.warp}, triangles)
            expect(spread).toBeLessThan(name === "earth-2" ? 1.0009 : 1.0013)
            expect(areaSpread(shape, triangles), "the gnomonic lattice's").toBeGreaterThan(1.8)
            for (const key of ["a", "b", "c"]) {
                for (const by of [-0.002, 0.002]) {
                    const nudged = {...lattice.warp, [key]: lattice.warp[key] + by}
                    expect(areaSpread({...shape, warp: nudged}, triangles), `${key} ${by > 0 ? "+" : ""}${by}`).toBeGreaterThan(spread)
                }
            }
        }, 120_000)
    }

    it("is in each new board's manifest as the generator defines it", () => {
        for (const {name, lattice} of [EARTH_2, SIEGE_1]) {
            const {manifest} = readBoard(boardDir(name))
            expect(manifest.lattice).toEqual(lattice)
            expect(BOARD_DEFINITIONS[name].lattice.order).toBe("canonical")
        }
    })
})
