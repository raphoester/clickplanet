// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {Season, SeasonBackend} from "../../backends/season.ts"
import {FakeSeasonBackend} from "../../backends/fakeSeasonBackend.ts"
import {Seasons, useSeason} from "./useSeason.ts"

const DAY = 24 * 60 * 60 * 1000

let latest: Season | undefined
let both: Seasons = {}

function Harness({backend}: {backend?: SeasonBackend}) {
    both = useSeason(backend)
    latest = both.season
    return null
}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("useSeason", () => {
    it("has no season with no backend", () => {
        render(<Harness/>)
        expect(latest).toBeUndefined()
    })

    it("holds the season the backend answers", async () => {
        const season = {number: 0, finaleStartsAt: Date.now() + DAY, endsAt: Date.now() + 2 * DAY}
        await act(async () => {
            render(<Harness backend={new FakeSeasonBackend(season)}/>)
        })
        expect(latest).toEqual(season)
    })

    it("has no season when the backend answers none", async () => {
        await act(async () => {
            render(<Harness backend={new FakeSeasonBackend(undefined)}/>)
        })
        expect(latest).toBeUndefined()
    })

    it("lets the season go at its end, even weeks away", async () => {
        vi.useFakeTimers({now: Date.UTC(2026, 9, 3, 12)})
        const season = {number: 0, finaleStartsAt: Date.UTC(2026, 9, 31, 21), endsAt: Date.UTC(2026, 9, 31, 23)}
        await act(async () => {
            render(<Harness backend={new FakeSeasonBackend(season)}/>)
        })
        expect(latest).toEqual(season)

        await act(async () => {
            vi.advanceTimersByTime(season.endsAt - Date.now() - 1000)
        })
        expect(latest).toEqual(season)

        await act(async () => {
            vi.advanceTimersByTime(1000)
        })
        expect(latest).toBeUndefined()
    })

    it("says nothing when the read fails", async () => {
        const error = vi.spyOn(console, "error").mockImplementation(() => {})
        await act(async () => {
            render(<Harness backend={{season: () => Promise.reject(new Error("down")), next: async () => undefined}}/>)
        })
        expect(latest).toBeUndefined()
        expect(error).toHaveBeenCalled()
    })

    it("waits between two seasons, then plays the next one from its start", async () => {
        vi.useFakeTimers({now: Date.UTC(2026, 9, 31, 22)})
        const zero = {number: 0, finaleStartsAt: Date.UTC(2026, 9, 31, 21), endsAt: Date.UTC(2026, 9, 31, 23)}
        const one = {number: 1, startsAt: Date.UTC(2026, 10, 1, 19), finaleStartsAt: Date.UTC(2026, 11, 5, 21), endsAt: Date.UTC(2026, 11, 5, 23)}
        await act(async () => {
            render(<Harness backend={new FakeSeasonBackend(zero, one)}/>)
        })
        expect(both).toEqual({season: zero, next: one})

        await act(async () => {
            vi.advanceTimersByTime(zero.endsAt - Date.now())
        })
        expect(both).toEqual({season: undefined, next: one})

        await act(async () => {
            vi.advanceTimersByTime(one.startsAt - Date.now())
        })
        expect(both).toEqual({season: one})
    })
})
