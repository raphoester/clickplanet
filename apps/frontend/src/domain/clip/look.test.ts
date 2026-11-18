import {describe, expect, it} from "vitest"
import {TileChange} from "./changes.ts"
import {Flip, flippedTo, flipsOf, minLandmassTiles, readableFlipTiles} from "./look.ts"
import {EARTH_1_SCALE, scaleOf} from "../boardScale.ts"
import {FIXTURE_1} from "../../testdata/boards.ts"

const landmassOf = [...Array(30).fill(1), ...Array(30).fill(2), ...Array(5).fill(3), 0]

function holding(owner: string, from: number, to: number): Map<number, string> {
    return new Map(Array.from({length: to - from + 1}, (_, i) => [from + i, owner]))
}

function taking(to: string | undefined, from: string | undefined, first: number, last: number, at = 0): TileChange[] {
    return Array.from({length: last - first + 1}, (_, i) => ({tile: first + i, from, to, at: at + i}))
}

describe("the landmasses that changed hands", () => {
    it("are the ones whose biggest holder changed", () => {
        const opening = new Map([...holding("pl", 1, 30), ...holding("fr", 31, 60)])

        expect(flipsOf(landmassOf, opening, taking("de", "pl", 1, 20), [1, 31], EARTH_1_SCALE)).toEqual([{landmass: 1, tiles: 30, holders: ["pl", "de"], tile: 1, readable: false}])
    })

    it("count a landmass taken and taken back", () => {
        const changes = [...taking("nl", "ps", 1, 25), ...taking("ps", "nl", 1, 25, 100)]

        expect(flipsOf(landmassOf, holding("ps", 1, 30), changes, [1], EARTH_1_SCALE)).toEqual([{landmass: 1, tiles: 30, holders: ["ps", "nl", "ps"], tile: 1, readable: false}])
    })

    it("leave out a landmass nothing touched, and a small island", () => {
        const changes = taking("de", "pl", 1, 65)

        expect(flipsOf(landmassOf, holding("pl", 1, 65), changes, [1, 61], EARTH_1_SCALE)).toHaveLength(1)
        expect(flipsOf(landmassOf, holding("pl", 1, 65), changes, [66], EARTH_1_SCALE)).toEqual([])
    })

    it("count a landmass emptied by bombs", () => {
        expect(flipsOf(landmassOf, holding("pl", 1, 30), taking(undefined, "pl", 1, 30), [1], EARTH_1_SCALE)).toEqual([{landmass: 1, tiles: 30, holders: ["pl", ""], tile: 1, readable: false}])
    })
})

describe("the landmasses whose painted flag switched to a flag", () => {
    it("are the ones another held at the start and it holds at the end, big enough to read: not one it held already, lost again, or an islet", () => {
        const flips: Flip[] = [
            {landmass: 1, tiles: 300, holders: ["fr", "pw"], tile: 1, readable: true},
            {landmass: 2, tiles: 150, holders: ["pw", "dz"], tile: 31, readable: true},
            {landmass: 3, tiles: 180, holders: ["ps", "pw", "ps"], tile: 61, readable: true},
            {landmass: 4, tiles: 120, holders: ["dz", "pw"], tile: 66, readable: true},
            {landmass: 5, tiles: 60, holders: ["dz", "pw"], tile: 70, readable: false},
        ]

        expect(flippedTo(flips, "pw").map(({landmass}) => landmass)).toEqual([1, 4])
    })
})

describe("the sizes a landmass's flip is read at", () => {
    it("are 20 and 100 tiles on earth-1", () => {
        expect(minLandmassTiles(EARTH_1_SCALE)).toBe(20)
        expect(readableFlipTiles(EARTH_1_SCALE)).toBe(100)
    })

    it("are as much ground on a coarser board", () => {
        const scale = scaleOf(FIXTURE_1)
        expect(minLandmassTiles(scale)).toBe(1)
        expect(readableFlipTiles(scale)).toBe(1)
        expect(flipsOf([1, 1, 2], new Map([[1, "pl"], [2, "pl"], [3, "fr"]]), taking("de", "pl", 1, 2), [1, 3], scale))
            .toEqual([{landmass: 1, tiles: 2, holders: ["pl", "de"], tile: 1, readable: true}])
    })
})
