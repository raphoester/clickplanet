import {describe, expect, it} from "vitest"
import {MineChange, mineMarksOf, TileMines} from "./mines.ts"

describe("mineMarksOf", () => {
    it("marks each tile as its last change left it", () => {
        expect(mineMarksOf([{tile: 1, mined: true}, {tile: 2, mined: true}, {tile: 1, mined: false}]))
            .toEqual({on: [2], off: [1]})
    })
})

describe("TileMines", () => {
    it("keeps a mine at the newest seq it is known at, and a change older than it leaves it", () => {
        const mines = new TileMines(10)
        const changes: MineChange[] = []

        mines.lay(4, 7, changes)
        mines.lay(4, 9, changes)
        mines.takeOff(4, 9, changes)
        expect(mines.has(4)).toBe(true)

        mines.takeOff(4, 10, changes)
        expect(mines.has(4)).toBe(false)
        expect(changes).toEqual([{tile: 4, mined: true}, {tile: 4, mined: false}])
    })

    it("lays no mine off the map", () => {
        const mines = new TileMines(10)
        const changes: MineChange[] = []

        mines.lay(11, 1, changes)
        mines.lay(0, 1, changes)

        expect(mines.count()).toBe(0)
        expect(changes).toEqual([])
    })
})
