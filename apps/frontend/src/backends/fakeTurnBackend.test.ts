import {describe, expect, it} from "vitest"
import {FakeTurnBackend, seasonRunningAt} from "./fakeTurnBackend.ts"

describe("seasonRunningAt", () => {
    it.each([
        ["after Season 0's dates", Date.UTC(2026, 10, 3, 16, 41)],
        ["a minute before the cutoff", Date.UTC(2026, 10, 3, 20, 59)],
        ["after the cutoff", Date.UTC(2026, 10, 3, 22, 30)],
        ["at midnight a year on", Date.UTC(2027, 10, 3)],
    ])("runs a day %s, so closeDay() debriefs it", (_, now) => {
        const turns = new FakeTurnBackend(seasonRunningAt(now), () => now)

        const today = turns.today()
        expect(today).toBeDefined()
        expect(today!.startsAt).toBeLessThanOrEqual(now)
        expect(today!.endsAt).toBeGreaterThan(now)
        expect(turns.endDay(now)).toEqual({kind: "day", season: 0, number: today!.number, roundId: today!.roundId, endedAt: now})
    })
})
