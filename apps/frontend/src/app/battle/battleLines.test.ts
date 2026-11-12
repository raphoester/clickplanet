import {describe, expect, it} from "vitest"
import {BattleToCome} from "../../backends/turn.ts"
import {Battle} from "../../domain/battle.ts"
import {translator} from "../../i18n/page.ts"
import {dayAndTimeRange, timeOfDay, timeRange} from "../season/dates.ts"
import {battleEndsLine, battleName, battleWhenLine, teaserClockLine} from "./battleLines.ts"

const MINUTE = 60_000
const HOUR = 60 * MINUTE

const at = (day: number, hour: number) => new Date(2026, 10, day, hour).getTime()

const battle: Battle = {entryId: "rush", since: at(5, 21), endsAt: at(5, 23), description: "", roundId: "rush"}
const toCome: BattleToCome = {entryId: "rush", startsAt: at(5, 22), endsAt: at(5, 23), description: "", times: 1, endsSeason: false, roundId: "rush"}

describe("battleName", () => {
    it("names the season's finale the Final Battle", () => {
        expect(battleName(battle)).toBe("Battle")
        expect(battleName({...battle, endsSeason: true})).toBe("Final Battle")
    })
})

describe("battleEndsLine", () => {
    it("says the time it ends, and the day when it is not today, in the player's time", () => {
        expect(battleEndsLine(battle, at(5, 21))).toBe(`Ends at ${timeOfDay(battle.endsAt)}`)
        const late = {...battle, endsAt: at(6, 1)}
        expect(battleEndsLine(late, at(5, 21))).toBe(`Ends ${translator().dateTime(late.endsAt, {weekday: "short"})} at ${timeOfDay(late.endsAt)}`)
    })
})

describe("battleWhenLine", () => {
    it("says today, tomorrow, or the date, with the hours in the player's time", () => {
        const hours = timeRange(toCome.startsAt, toCome.endsAt)
        expect(battleWhenLine(toCome, at(5, 10))).toBe(`Today · ${hours}`)
        expect(battleWhenLine(toCome, at(4, 23))).toBe(`Tomorrow · ${hours}`)
        expect(battleWhenLine(toCome, at(3, 22))).toBe(dayAndTimeRange(toCome.startsAt, toCome.endsAt))
    })

    it("says what it is worth when it is more than a day's points", () => {
        expect(battleWhenLine({...toCome, times: 2}, at(5, 10))).toBe(`Today · ${timeRange(toCome.startsAt, toCome.endsAt)} · Points ×2`)
    })
})

describe("teaserClockLine", () => {
    it("counts to the start before the battle, to its end while it plays", () => {
        expect(teaserClockLine(toCome, toCome.startsAt - 12 * HOUR)).toBe("Starts in 12h 00m 00s")
        expect(teaserClockLine(toCome, toCome.startsAt)).toBe("Ends in 1h 00m 00s")
        expect(teaserClockLine(toCome, toCome.startsAt + 18 * MINUTE)).toBe("Ends in 42m 00s")
    })
})
