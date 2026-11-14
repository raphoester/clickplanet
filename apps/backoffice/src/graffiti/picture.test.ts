import {describe, expect, it} from "vitest"
import {Cell, pictureOf} from "./picture.ts"
import {patch} from "./testPlaces.ts"

const middleOf = (cell: Cell) => [
    cell.corners.reduce((sum, [x]) => sum + x, 0) / cell.corners.length,
    cell.corners.reduce((sum, [, y]) => sum + y, 0) / cell.corners.length,
]

// A plus of five tiles on its centre.
const PLUS = [[0, 0], [1, 0], [-1, 0], [0, 1], [0, -1]]

describe("pictureOf", () => {
    it("draws each tile of the drawing as a hexagon over the land around it, on the drawing's middle", () => {
        const {places, idOf} = patch(48.85, 2.35)
        const picture = pictureOf(places, PLUS.map(([i, j]) => idOf(i, j)))!

        const drawn = picture.cells.filter(cell => cell.drawn)
        expect(drawn.map(cell => cell.tile).sort()).toEqual(PLUS.map(([i, j]) => idOf(i, j)).sort())
        expect(picture.cells.length).toBeGreaterThan(drawn.length * 4)
        expect(picture.cells.every(cell => cell.corners.length === 6)).toBe(true)
        expect(picture.latitude).toBeCloseTo(48.85, 1)
        expect(picture.longitude).toBeCloseTo(2.35, 1)
        // The drawing last, painted over the land.
        expect(picture.cells.slice(-drawn.length).every(cell => cell.drawn)).toBe(true)
    })

    it("frames the drawing with a margin of land, north up and east right", () => {
        const {places, idOf} = patch(-26.5, -51.5)
        const picture = pictureOf(places, PLUS.map(([i, j]) => idOf(i, j)))!
        const at = (i: number, j: number) => middleOf(picture.cells.find(cell => cell.tile === idOf(i, j))!)

        const [centreX, centreY] = at(0, 0)
        expect(centreX).toBeCloseTo(0, 5)
        expect(centreY).toBeCloseTo(0, 5)
        expect(at(1, 0)[0]).toBeGreaterThan(centreX)
        expect(at(0, 1)[1]).toBeLessThan(centreY)
        // One step out of five and a bit: the plus is a tile from its middle, with four tiles of margin.
        expect(at(1, 0)[0]).toBeGreaterThan(0.15)
        expect(at(1, 0)[0]).toBeLessThan(0.25)
    })

    it("keeps a tile on the coast whole, where the sea leaves it neighbours short", () => {
        const {places, idOf} = patch(10, 20, 6)
        const shore = [[6, 0], [5, 0], [6, -1]]
        const picture = pictureOf(places, shore.map(([i, j]) => idOf(i, j)))!

        const drawn = picture.cells.filter(cell => cell.drawn)
        expect(drawn).toHaveLength(3)
        expect(drawn.every(cell => cell.corners.length === 6)).toBe(true)
    })

    it("draws across the antimeridian and next to a pole as anywhere else", () => {
        for (const [latitude, longitude] of [[0, 179.9], [89.8, 0], [-89.9, 120]]) {
            const {places, idOf} = patch(latitude, longitude)
            const picture = pictureOf(places, PLUS.map(([i, j]) => idOf(i, j)))!

            expect(picture.cells.filter(cell => cell.drawn)).toHaveLength(5)
            expect(picture.cells.length).toBeGreaterThan(20)
        }
    })

    it("leaves out an id the map does not hold, and draws nothing when none is left", () => {
        const {places, idOf} = patch(0, 0, 3)

        expect(pictureOf(places, [0, 9999])).toBeUndefined()
        expect(pictureOf(places, [idOf(0, 0), 9999])!.cells.filter(cell => cell.drawn)).toHaveLength(1)
    })
})
