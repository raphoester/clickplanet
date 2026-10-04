// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {Season, SeasonBackend} from "../../backends/season.ts"
import {FakeSeasonBackend} from "../../backends/fakeSeasonBackend.ts"
import {useSeason} from "./useSeason.ts"

const DAY = 24 * 60 * 60 * 1000

let latest: Season | undefined

function Harness({backend}: {backend?: SeasonBackend}) {
    latest = useSeason(backend)
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
            render(<Harness backend={{season: () => Promise.reject(new Error("down"))}}/>)
        })
        expect(latest).toBeUndefined()
        expect(error).toHaveBeenCalled()
    })
})
