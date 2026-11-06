import {describe, expect, it} from "vitest"
import {BattleToCome, PLAIN_TURN, Turn} from "../backends/turn.ts"
import {BATTLE_TEASER_EVERY_MS, battleTeaserDue, battleWhenLine, playingOf, teasedOf, teaserClockLine, teaserKeyOf} from "./battleTeaser.ts"

const MINUTE = 60_000
const HOUR = 60 * MINUTE

// Thu 5 Nov 2026, 08:00 UTC: 10:00 in Sofia.
const now = Date.UTC(2026, 10, 5, 8)
const battle: BattleToCome = {
    entryId: "tonight", startsAt: Date.UTC(2026, 10, 5, 20), endsAt: Date.UTC(2026, 10, 5, 21),
    description: "", times: 2, endsSeason: false, roundId: "tonight",
}
const finale: BattleToCome = {...battle, entryId: "", endsSeason: true, roundId: "season-1/finale"}

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
        const teased = {key: "tonight", at: now}
        expect(battleTeaserDue(battle, teased, now + HOUR, true)).toBe(false)
        expect(battleTeaserDue(battle, teased, now + BATTLE_TEASER_EVERY_MS - 1, true)).toBe(false)
        expect(battleTeaserDue(battle, teased, now + BATTLE_TEASER_EVERY_MS, true)).toBe(true)
    })

    it("tells of another battle at once", () => {
        expect(battleTeaserDue(battle, {key: "yesterday", at: now - MINUTE}, now, true)).toBe(true)
    })

    it("teases the Final Battle by its round", () => {
        expect(teaserKeyOf(finale)).toBe("season-1/finale")
        expect(battleTeaserDue(finale, null, now, true)).toBe(true)
        expect(battleTeaserDue(finale, {key: "season-1/finale", at: now}, now + HOUR, true)).toBe(false)
    })

    it("waits for a newcomer to play, and for a battle to come", () => {
        expect(battleTeaserDue(battle, null, now, false)).toBe(false)
        expect(battleTeaserDue(undefined, null, now, true)).toBe(false)
    })
})

describe("teasedOf", () => {
    it("reads what was kept, and nothing from anything else", () => {
        expect(teasedOf(JSON.stringify({key: "tonight", at: now}))).toEqual({key: "tonight", at: now})
        expect(teasedOf(null)).toBeNull()
        expect(teasedOf("tonight")).toBeNull()
        expect(teasedOf("null")).toBeNull()
        expect(teasedOf(JSON.stringify({key: 3, at: now}))).toBeNull()
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

describe("playingOf", () => {
    const turn: Turn = {
        kind: "battle", since: battle.startsAt, endsAt: battle.endsAt, description: "Rush.", entryId: "tonight",
        roundId: "tonight", times: 2, endsSeason: false,
    }

    it("is the battle that plays, in the shape of one to come", () => {
        expect(playingOf(turn)).toEqual({...battle, description: "Rush."})
    })

    it("is the season's finale, keyed by its round", () => {
        const playing = playingOf({...turn, entryId: "", roundId: "season-1/finale", endsSeason: true})
        expect(playing && teaserKeyOf(playing)).toBe("season-1/finale")
    })

    it("is none for a plain turn, or none at all", () => {
        expect(playingOf(PLAIN_TURN)).toBeUndefined()
        expect(playingOf(undefined)).toBeUndefined()
    })
})

describe("teaserClockLine", () => {
    it("counts to the start before the battle, to its end while it plays", () => {
        expect(teaserClockLine(battle, now)).toBe("Starts in 12h 00m 00s")
        expect(teaserClockLine(battle, battle.startsAt)).toBe("Ends in 1h 00m 00s")
        expect(teaserClockLine(battle, battle.startsAt + 18 * MINUTE)).toBe("Ends in 42m 00s")
    })
})
