import {describe, expect, it} from "vitest"
import {CapitalMarks} from "./capitalMarks.ts"

const CAPITALS = new Map([[10, "fr"], [20, "de"], [30, "jp"]])

describe("CapitalMarks", () => {
    it("marks each capital its own country holds", () => {
        const owners = new Map([[10, "fr"], [20, "fr"]])
        const marks = new CapitalMarks(CAPITALS, (tile) => owners.get(tile))

        expect(marks.follow()).toEqual({on: [10], off: []})
    })

    it("puts out a capital another country takes, and lights it again when its own takes it back", () => {
        const owners = new Map([[10, "fr"], [20, "de"]])
        const marks = new CapitalMarks(CAPITALS, (tile) => owners.get(tile))
        marks.follow()

        owners.set(20, "fr")
        expect(marks.touch([20, 21])).toEqual({on: [], off: [20]})

        owners.set(20, "de")
        expect(marks.touch([20])).toEqual({on: [20], off: []})
    })

    it("judges only the capitals among the tiles it is told of", () => {
        const owners = new Map([[11, "fr"]])
        const marks = new CapitalMarks(CAPITALS, (tile) => owners.get(tile))
        marks.follow()

        expect(marks.touch([11, 12])).toEqual({on: [], off: []})
    })

    it("puts out every mark when dropped", () => {
        const owners = new Map([[10, "fr"], [30, "jp"]])
        const marks = new CapitalMarks(CAPITALS, (tile) => owners.get(tile))
        marks.follow()

        const {on, off} = marks.drop()

        expect(on).toEqual([])
        expect(off.sort()).toEqual([10, 30])
    })
})
