import {describe, expect, it} from "vitest"
import {FlagMarks, MarkChange, marksOf} from "./flagMarks.ts"

describe("marksOf", () => {
    it("marks each tile as its last change left it", () => {
        expect(marksOf([{tile: 1, marked: true}, {tile: 2, marked: true}, {tile: 1, marked: false}]))
            .toEqual({on: [2], off: [1]})
    })
})

describe("FlagMarks", () => {
    it("keeps a mark at the newest seq it is known at, and a change older than it leaves it", () => {
        const marks = new FlagMarks(10)
        const changes: MarkChange[] = []

        marks.lay(4, 7, changes)
        marks.lay(4, 9, changes)
        marks.takeOff(4, 9, changes)
        expect(marks.has(4)).toBe(true)

        marks.takeOff(4, 10, changes)
        expect(marks.has(4)).toBe(false)
        expect(changes).toEqual([{tile: 4, marked: true}, {tile: 4, marked: false}])
    })

    it("lays no mark off the map", () => {
        const marks = new FlagMarks(10)
        const changes: MarkChange[] = []

        marks.lay(11, 1, changes)
        marks.lay(0, 1, changes)

        expect(marks.count()).toBe(0)
        expect(changes).toEqual([])
    })
})
