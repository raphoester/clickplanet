import {describe, expect, it} from "vitest"
import {TileChange} from "./changes.ts"
import {Flip, flippedTo, flipsOf, lookOf, TILES_ZOOM} from "./look.ts"

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

        expect(flipsOf(landmassOf, opening, taking("de", "pl", 1, 20), [1, 31])).toEqual([{landmass: 1, tiles: 30, holders: ["pl", "de"], tile: 1}])
    })

    it("count a landmass taken and taken back", () => {
        const changes = [...taking("nl", "ps", 1, 25), ...taking("ps", "nl", 1, 25, 100)]

        expect(flipsOf(landmassOf, holding("ps", 1, 30), changes, [1])).toEqual([{landmass: 1, tiles: 30, holders: ["ps", "nl", "ps"], tile: 1}])
    })

    it("leave out a landmass nothing touched, and a small island", () => {
        const changes = taking("de", "pl", 1, 65)

        expect(flipsOf(landmassOf, holding("pl", 1, 65), changes, [1, 61])).toHaveLength(1)
        expect(flipsOf(landmassOf, holding("pl", 1, 65), changes, [66])).toEqual([])
    })

    it("count a landmass emptied by bombs", () => {
        expect(flipsOf(landmassOf, holding("pl", 1, 30), taking(undefined, "pl", 1, 30), [1])).toEqual([{landmass: 1, tiles: 30, holders: ["pl", ""], tile: 1}])
    })
})

describe("the look of a clip", () => {
    const flip = (tiles: number): Flip => ({landmass: 1, tiles, holders: ["fr", "ps"], tile: 1})

    it("is painted flags when several big landmasses changed hands over a wide front", () => {
        expect(lookOf([flip(1500), flip(900)], 1.5)).toBe("flags")
    })

    it("dives into the tiles when a wide front changed one big landmass, or some smaller ones", () => {
        expect(lookOf([flip(2500)], 1.5)).toBe("dive")
        expect(lookOf([flip(1000), flip(600)], 1.5)).toBe("dive")
    })

    it("dives when nothing big enough to see from far changed hands", () => {
        expect(lookOf([flip(60), flip(40), flip(30)], 1.5)).toBe("dive")
        expect(lookOf([], 1.5)).toBe("dive")
    })

    it("dives rather than paint flags all along when the fight fits a view close enough to read the tiles", () => {
        expect(lookOf([flip(1500), flip(900)], TILES_ZOOM)).toBe("dive")
    })
})

describe("the landmasses whose painted flag switched to a flag", () => {
    it("are the ones another held at the start and it holds at the end, big enough to read: not one it held already, lost again, or an islet", () => {
        const flips: Flip[] = [
            {landmass: 1, tiles: 300, holders: ["fr", "pw"], tile: 1},
            {landmass: 2, tiles: 150, holders: ["pw", "dz"], tile: 31},
            {landmass: 3, tiles: 180, holders: ["ps", "pw", "ps"], tile: 61},
            {landmass: 4, tiles: 120, holders: ["dz", "pw"], tile: 66},
            {landmass: 5, tiles: 60, holders: ["dz", "pw"], tile: 70},
        ]

        expect(flippedTo(flips, "pw").map(({landmass}) => landmass)).toEqual([1, 4])
    })
})
