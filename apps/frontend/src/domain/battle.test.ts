import {describe, expect, it} from "vitest"
import {Turn} from "../backends/turn.ts"
import {
    Battle,
    battleDue,
    battleEndsLine,
    battleKeyOf,
    battleLeft,
    battleOf,
    nextBattleOf,
} from "./battle.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE

const now = Date.UTC(2026, 9, 31, 21, 30)
const turn: Turn = {kind: "battle", since: now - 30 * MINUTE, endsAt: now + 2 * HOUR, description: "Triple points.", entryId: "entry-final", roundId: "entry-final", times: 3, endsSeason: false}
const battle: Battle = {entryId: "entry-final", since: turn.since!, endsAt: turn.endsAt!, description: "Triple points.", roundId: "entry-final", endsSeason: false}

describe("battleOf", () => {
    it("is the battle the turn plays", () => {
        expect(battleOf(turn)).toEqual(battle)
    })

    it("is nothing for a plain turn, no turn, or a battle with no end", () => {
        expect(battleOf({kind: "plain", description: "", entryId: "", roundId: "", times: 0, endsSeason: false})).toBeUndefined()
        expect(battleOf(undefined)).toBeUndefined()
        expect(battleOf({kind: "battle", description: "", entryId: "entry-final", roundId: "entry-final", times: 3, endsSeason: false})).toBeUndefined()
    })
})

describe("nextBattleOf", () => {
    const soon = {entryId: "soon", startsAt: now + HOUR, endsAt: now + 3 * HOUR, description: "Triple points.", times: 3, endsSeason: false, roundId: "soon"}
    const later = {...soon, entryId: "later", startsAt: now + 2 * HOUR}

    it("is the first battle that has not started", () => {
        expect(nextBattleOf([soon, later], now)).toBe(soon)
        expect(nextBattleOf([soon, later], soon.startsAt)).toBe(later)
    })

    it("is nothing with no battle to come", () => {
        expect(nextBattleOf([], now)).toBeUndefined()
        expect(nextBattleOf([soon], later.startsAt)).toBeUndefined()
    })
})

describe("the battle's key", () => {
    it("is its entry's id", () => {
        expect(battleKeyOf(battle)).toBe("entry-final")
    })

    it("is due until it was seen", () => {
        expect(battleDue(battle, null)).toBe(true)
        expect(battleDue(battle, "entry-before")).toBe(true)
        expect(battleDue(battle, battleKeyOf(battle))).toBe(false)
        expect(battleDue(undefined, null)).toBe(false)
    })
})

describe("the battle's clock", () => {
    it("counts down to the end", () => {
        expect(battleLeft(battle, now)).toBe("2h 00m 00s")
        expect(battleLeft(battle, now + HOUR + 36 * MINUTE + 55 * SECOND)).toBe("23m 05s")
    })

    it("stops at zero", () => {
        expect(battleLeft(battle, battle.endsAt + MINUTE)).toBe("00m 00s")
    })

    it("says when the battle ends, in the player's time", () => {
        expect(battleEndsLine(battle, now, "UTC")).toBe("Ends at 23:30")
        expect(battleEndsLine(battle, now, "Europe/Sofia")).toBe("Ends Sun at 01:30")
    })
})
