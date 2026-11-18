import {describe, expect, it} from "vitest"
import {PlayerLevel} from "../backends/progression.ts"
import {barOf, gainedOn, newerLevel, nextUnlock, unlockedBy} from "./level.ts"

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

    it("keeps the XP a gain showed over a read sent before it", () => {
        const gained = {...twelve, xp: 20_100}
        expect(newerLevel(gained, twelve)).toBe(gained)
    })
})

describe("gainedOn", () => {
    it("moves the XP on the level shown", () => {
        expect(gainedOn(twelve, {...twelve, xp: 20_100})).toEqual({...twelve, xp: 20_100})
    })

    it("leaves the level to the levelReached that follows, the bar full meanwhile", () => {
        const thirteen = {level: 13, xp: 24_200, levelXp: 24_192, nextLevelXp: 30_758}
        const shown = gainedOn(twelve, thirteen)

        expect(shown).toEqual({...twelve, xp: 24_200})
        expect(barOf(shown!).fill).toBe(1)
    })

    it("shows no level the page had none of", () => {
        expect(gainedOn(undefined, twelve)).toBeUndefined()
    })

    it("keeps what it shows when the gain carries no level", () => {
        expect(gainedOn(twelve, undefined)).toBe(twelve)
    })
})

const LADDER = {spreadClicks: 2, shields: 3, encloseClicks: 5, bomb: 7, virus: 10, mine: 13, rally: 15}

describe("nextUnlock", () => {
    it("is the lowest rung above the level", () => {
        expect(nextUnlock(LADDER, 1)).toEqual({kind: "spreadClicks", level: 2})
        expect(nextUnlock(LADDER, 10)).toEqual({kind: "mine", level: 13})
        expect(nextUnlock(LADDER, 12)).toEqual({kind: "mine", level: 13})
    })

    it("is none once every kind is unlocked", () => {
        expect(nextUnlock(LADDER, 15)).toBeUndefined()
        expect(nextUnlock(LADDER, 25)).toBeUndefined()
        expect(nextUnlock({}, 1)).toBeUndefined()
    })
})

describe("unlockedBy", () => {
    it("is the rung a raise reached", () => {
        expect(unlockedBy(LADDER, 12, 13)).toEqual([{kind: "mine", level: 13}])
    })

    it("is every rung a jump passed, in order", () => {
        expect(unlockedBy(LADDER, 6, 11)).toEqual([{kind: "bomb", level: 7}, {kind: "virus", level: 10}])
    })

    it("is none for a level that unlocks nothing", () => {
        expect(unlockedBy(LADDER, 15, 16)).toEqual([])
        expect(unlockedBy(LADDER, 13, 14)).toEqual([])
    })
})
