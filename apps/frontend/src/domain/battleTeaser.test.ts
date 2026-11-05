import {describe, expect, it} from "vitest"
import {BattleToCome} from "../backends/turn.ts"
import {BATTLE_TEASER_EVERY_MS, battleTeaserDue, battleWhenLine, teasedOf, teaserCardsOf} from "./battleTeaser.ts"

const MINUTE = 60_000
const HOUR = 60 * MINUTE

// Thu 5 Nov 2026, 08:00 UTC: 10:00 in Sofia.
const now = Date.UTC(2026, 10, 5, 8)
const battle: BattleToCome = {
    entryId: "tonight", startsAt: Date.UTC(2026, 10, 5, 20), endsAt: Date.UTC(2026, 10, 5, 21),
    description: "", times: 2, endsSeason: false, roundId: "tonight",
}

describe("battleTeaserDue", () => {
    it("teases the next battle in the day before it starts", () => {
        expect(battleTeaserDue(battle, null, now, true)).toBe(true)
        expect(battleTeaserDue(battle, null, battle.startsAt - 24 * HOUR, true)).toBe(true)
        expect(battleTeaserDue(battle, null, battle.startsAt - 24 * HOUR - MINUTE, true)).toBe(false)
    })

    it("stops once the battle started", () => {
        expect(battleTeaserDue(battle, null, battle.startsAt - 1, true)).toBe(true)
        expect(battleTeaserDue(battle, null, battle.startsAt, true)).toBe(false)
    })

    it("tells a browser again once the hours passed", () => {
        const teased = {entryId: "tonight", at: now}
        expect(battleTeaserDue(battle, teased, now + HOUR, true)).toBe(false)
        expect(battleTeaserDue(battle, teased, now + BATTLE_TEASER_EVERY_MS - 1, true)).toBe(false)
        expect(battleTeaserDue(battle, teased, now + BATTLE_TEASER_EVERY_MS, true)).toBe(true)
    })

    it("tells of another battle at once", () => {
        expect(battleTeaserDue(battle, {entryId: "yesterday", at: now - MINUTE}, now, true)).toBe(true)
    })

    it("waits for a newcomer to play", () => {
        expect(battleTeaserDue(battle, null, now, false)).toBe(false)
    })

    it("leaves the season's finale to its own teaser, and no battle is none", () => {
        expect(battleTeaserDue({...battle, entryId: "", endsSeason: true, roundId: "season-1/finale"}, null, now, true)).toBe(false)
        expect(battleTeaserDue(undefined, null, now, true)).toBe(false)
    })
})

describe("teasedOf", () => {
    it("reads what was kept, and nothing from anything else", () => {
        expect(teasedOf(JSON.stringify({entryId: "tonight", at: now}))).toEqual({entryId: "tonight", at: now})
        expect(teasedOf(null)).toBeNull()
        expect(teasedOf("tonight")).toBeNull()
        expect(teasedOf("null")).toBeNull()
        expect(teasedOf(JSON.stringify({entryId: 3, at: now}))).toBeNull()
    })
})

describe("battleWhenLine", () => {
    it("says the day and the hours in the player's time", () => {
        expect(battleWhenLine(battle, now, "Europe/Sofia")).toBe("Today · 22:00 – 23:00")
        expect(battleWhenLine(battle, now, "UTC")).toBe("Today · 20:00 – 21:00")
        expect(battleWhenLine(battle, now - 12 * HOUR, "Europe/Sofia")).toBe("Tomorrow · 22:00 – 23:00")
        expect(battleWhenLine(battle, now - 48 * HOUR, "Europe/Sofia")).toBe("Thu 5 Nov · 22:00 – 23:00")
    })

    it("counts the day where the player is", () => {
        expect(battleWhenLine(battle, Date.UTC(2026, 10, 5, 23), "America/Los_Angeles")).toBe("Today · 12:00 – 13:00")
        expect(battleWhenLine(battle, Date.UTC(2026, 10, 4, 23), "Asia/Tokyo")).toBe("Tomorrow · 05:00 – 06:00")
    })
})

describe("teaserCardsOf", () => {
    it("says how long, what it is worth, and when to be there", () => {
        expect(teaserCardsOf(battle, "Europe/Sofia")).toEqual([
            {kind: "war", head: "1 hour of war", text: "Every country fights at once. Keep your bombs and shields for it."},
            {kind: "points", head: "Points ×2", text: "The countries holding the most ground at 23:00 score the points of 2 days."},
            {kind: "rally", head: "Be there at 22:00", text: "Bring your friends. Your flag needs every click."},
        ])
    })

    it("promises no multiple at ×1", () => {
        expect(teaserCardsOf({...battle, times: 1}, "UTC")[1]).toEqual(
            {kind: "points", head: "A round of its own", text: "The countries holding the most ground at 21:00 score its points."})
    })

    it("says a length that is not whole hours", () => {
        expect(teaserCardsOf({...battle, endsAt: battle.startsAt + 90 * MINUTE})[0].head).toBe("90 minutes of war")
        expect(teaserCardsOf({...battle, endsAt: battle.startsAt + 2 * HOUR})[0].head).toBe("2 hours of war")
    })
})
