import {describe, expect, it} from "vitest"
import {Turn} from "../backends/turn.ts"
import {
    Battle,
    battleDue,
    battleEndsLine,
    battleKeyOf,
    battleLeft,
    battleOf,
    finaleLengthLine,
    finaleNear,
    finaleOf,
    finaleTeaserDue,
    finaleWhenLine,
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

describe("finaleOf", () => {
    const plain: Turn = {kind: "plain", description: "", entryId: "", roundId: "", times: 0, endsSeason: false}
    const finale = {entryId: "", startsAt: now + HOUR, endsAt: now + 3 * HOUR, description: "Triple points.", times: 3, endsSeason: true, roundId: "season-0/finale"}
    const battleBefore = {...finale, entryId: "before", startsAt: now + 10 * MINUTE, endsAt: now + 40 * MINUTE, endsSeason: false, roundId: "before"}

    it("is the season's finale to come, never another battle", () => {
        expect(finaleOf([battleBefore, finale], plain, now)).toEqual({
            roundId: "season-0/finale", startsAt: finale.startsAt, endsAt: finale.endsAt, times: 3, description: "Triple points.", live: false,
        })
        expect(finaleOf([battleBefore], plain, now)).toBeUndefined()
    })

    it("is live from its start, before its turn arrives", () => {
        expect(finaleOf([finale], plain, finale.startsAt)?.live).toBe(true)
    })

    it("is the turn while the finale plays", () => {
        const playing: Turn = {...turn, since: finale.startsAt, endsAt: finale.endsAt, entryId: "", roundId: finale.roundId, times: 5, endsSeason: true}
        expect(finaleOf([], playing, now + 2 * HOUR)).toEqual({
            roundId: "season-0/finale", startsAt: finale.startsAt, endsAt: finale.endsAt, times: 5, description: "Triple points.", live: true,
        })
        expect(finaleOf([], turn, now)).toBeUndefined()
    })

    it("is nothing once it ended", () => {
        expect(finaleOf([finale], plain, finale.endsAt)).toBeUndefined()
    })

    it("says when it plays, in the player's time", () => {
        const found = finaleOf([finale], plain, now)!
        expect(finaleWhenLine(found, "UTC")).toBe("Sat 31 Oct · 22:30 – 00:30")
        expect(finaleWhenLine(found, "Europe/Sofia")).toBe("Sun 1 Nov · 00:30 – 02:30")
    })

    it("is near in the two days before its start, and no longer once it plays", () => {
        const at = (startsAt: number) => finaleOf([{...finale, startsAt, endsAt: startsAt + 2 * HOUR}], plain, now)
        expect(finaleNear(at(now + 48 * HOUR), now)).toBe(true)
        expect(finaleNear(at(now + 48 * HOUR + SECOND), now)).toBe(false)
        expect(finaleNear(at(now), now)).toBe(false)
        expect(finaleNear(undefined, now)).toBe(false)
    })

    it("is teased once per browser, while it is near", () => {
        const found = finaleOf([finale], plain, now)
        expect(finaleTeaserDue(found, null, now)).toBe(true)
        expect(finaleTeaserDue(found, "season-0/day-23", now)).toBe(true)
        expect(finaleTeaserDue(found, "season-0/finale", now)).toBe(false)
        expect(finaleTeaserDue(found, null, now - 48 * HOUR)).toBe(false)
    })

    it("says how long it lasts", () => {
        const lasting = (minutes: number) => finaleLengthLine({...finaleOf([finale], plain, now)!, endsAt: finale.startsAt + minutes * MINUTE})
        expect(lasting(120)).toBe("2 hours")
        expect(lasting(60)).toBe("1 hour")
        expect(lasting(90)).toBe("90 minutes")
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
