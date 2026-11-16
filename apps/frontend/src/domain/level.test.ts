import {describe, expect, it} from "vitest"
import {PlayerLevel} from "../backends/progression.ts"
import {barOf, newerLevel} from "./level.ts"

const twelve: PlayerLevel = {level: 12, xp: 20_000, levelXp: 18_634, nextLevelXp: 24_192}

describe("barOf", () => {
    it("fills the bar with the XP gained on the level", () => {
        expect(barOf(twelve)).toEqual({fill: 1366 / 5558, gained: 1366, span: 5558, left: 4192})
    })

    it("shows it full while the XP waits past the next threshold for the raise", () => {
        expect(barOf({...twelve, xp: 30_000})).toEqual({fill: 1, gained: 5558, span: 5558, left: 0})
    })

    it("shows it empty when the XP sits under the level's own threshold", () => {
        expect(barOf({...twelve, xp: 10_000})).toEqual({fill: 0, gained: 0, span: 5558, left: 5558})
    })

    it("shows a level with no span as full", () => {
        expect(barOf({level: 3, xp: 5, levelXp: 5, nextLevelXp: 5}).fill).toBe(1)
    })
})

describe("newerLevel", () => {
    it("takes what the server answers", () => {
        expect(newerLevel(undefined, twelve)).toBe(twelve)
        expect(newerLevel(twelve, {...twelve, xp: 21_000})).toEqual({...twelve, xp: 21_000})
    })

    it("keeps a level reached over a read sent before it", () => {
        const thirteen = {level: 13, xp: 24_200, levelXp: 24_192, nextLevelXp: 30_758}
        expect(newerLevel(thirteen, twelve)).toBe(thirteen)
    })

    it("keeps what it shows when a read answers none", () => {
        expect(newerLevel(twelve, undefined)).toBe(twelve)
    })
})
