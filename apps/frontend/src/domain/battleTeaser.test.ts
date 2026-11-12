import {describe, expect, it} from "vitest"
import {BattleToCome, PLAIN_TURN, Turn} from "../backends/turn.ts"
import {BATTLE_TEASER_EVERY_MS, BATTLE_TEASER_LAST_CALL_MS, battleDayOf, battleTeaserDue, playingOf, teasedOf, teaserClockOf, teaserKeyOf} from "./battleTeaser.ts"

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

    it("tells a browser once more in the battle's last hour", () => {
        const lastCall = battle.startsAt - BATTLE_TEASER_LAST_CALL_MS
        const teased = {key: "tonight", at: lastCall - HOUR}
        expect(battleTeaserDue(battle, teased, lastCall - 1, true)).toBe(false)
        expect(battleTeaserDue(battle, teased, lastCall, true)).toBe(true)
        expect(battleTeaserDue(battle, {key: "tonight", at: lastCall}, lastCall + 30 * MINUTE, true)).toBe(false)
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

describe("battleDayOf", () => {
    it("is the day it starts on, from now, in the player's time", () => {
        const evening = {...battle, startsAt: new Date(2026, 10, 5, 22).getTime(), endsAt: new Date(2026, 10, 5, 23).getTime()}
        expect(battleDayOf(evening, new Date(2026, 10, 5, 10).getTime())).toBe("today")
        expect(battleDayOf(evening, new Date(2026, 10, 4, 23).getTime())).toBe("tomorrow")
        expect(battleDayOf(evening, new Date(2026, 10, 3, 22).getTime())).toBe("later")
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

describe("teaserClockOf", () => {
    it("counts to the start before the battle, to its end while it plays", () => {
        expect(teaserClockOf(battle, now)).toEqual({to: "start", countdown: {days: 0, hours: 12, minutes: 0, seconds: 0}})
        expect(teaserClockOf(battle, battle.startsAt)).toEqual({to: "end", countdown: {days: 0, hours: 1, minutes: 0, seconds: 0}})
        expect(teaserClockOf(battle, battle.startsAt + 18 * MINUTE)).toEqual({to: "end", countdown: {days: 0, hours: 0, minutes: 42, seconds: 0}})
    })
})
