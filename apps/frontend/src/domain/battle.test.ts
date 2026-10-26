import {describe, expect, it} from "vitest"
import {Turn} from "../backends/turn.ts"
import {
    Battle,
    battleDue,
    battleEndsLine,
    battleKeyOf,
    battleLeft,
    battleOf,
    nextReadAt,
    TURN_REREAD_MS,
} from "./battle.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE

const now = Date.UTC(2026, 9, 31, 21, 30)
const turn: Turn = {kind: "battle", since: now - 30 * MINUTE, endsAt: now + 2 * HOUR, description: "Triple points."}
const battle: Battle = {since: turn.since!, endsAt: turn.endsAt!, description: "Triple points."}

describe("battleOf", () => {
    it("is the battle the turn plays", () => {
        expect(battleOf(turn)).toEqual(battle)
    })

    it("is nothing for a plain turn, no turn, or a battle with no end", () => {
        expect(battleOf({kind: "plain", description: ""})).toBeUndefined()
        expect(battleOf(undefined)).toBeUndefined()
        expect(battleOf({kind: "battle", description: ""})).toBeUndefined()
    })
})

describe("nextReadAt", () => {
    it("reads again in a minute", () => {
        expect(nextReadAt(undefined, now)).toBe(now + TURN_REREAD_MS)
        expect(nextReadAt({kind: "plain", description: ""}, now)).toBe(now + TURN_REREAD_MS)
        expect(nextReadAt(turn, now)).toBe(now + TURN_REREAD_MS)
    })

    it("reads again at the entry's end when it is sooner", () => {
        expect(nextReadAt({...turn, endsAt: now + 10 * SECOND}, now)).toBe(now + 10 * SECOND)
    })

    it("does not wait on an end already passed", () => {
        expect(nextReadAt({...turn, endsAt: now - SECOND}, now)).toBe(now + TURN_REREAD_MS)
    })
})

describe("the battle's key", () => {
    it("is its start and its end", () => {
        expect(battleKeyOf(battle)).toBe(`${battle.since}:${battle.endsAt}`)
    })

    it("is due until it was seen", () => {
        expect(battleDue(battle, null)).toBe(true)
        expect(battleDue(battle, "1:2")).toBe(true)
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
