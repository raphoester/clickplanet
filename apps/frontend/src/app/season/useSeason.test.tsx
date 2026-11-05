// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {Season} from "../../backends/season.ts"
import {useSeason} from "./useSeason.ts"

let latest: Season | undefined

function Harness({current}: {current?: Season}) {
    latest = useSeason(current)
    return null
}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("useSeason", () => {
    it("has no season while none runs", () => {
        render(<Harness/>)
        expect(latest).toBeUndefined()
    })

    it("holds the season the timeline says runs, and its moved end", () => {
        const season = {number: 0, startsAt: Date.now() - 1000, endsAt: Date.now() + 1000 * 60, bookId: "season-0"}
        const {rerender} = render(<Harness current={season}/>)
        expect(latest).toEqual(season)

        const moved = {...season, endsAt: season.endsAt + 1000 * 60}
        rerender(<Harness current={moved}/>)
        expect(latest).toEqual(moved)
    })

    it("lets the season go at its end, even weeks away", () => {
        vi.useFakeTimers({now: Date.UTC(2026, 9, 3, 12)})
        const season = {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt: Date.UTC(2026, 9, 31, 23), bookId: "season-0"}
        render(<Harness current={season}/>)
        expect(latest).toEqual(season)

        act(() => {
            vi.advanceTimersByTime(season.endsAt - Date.now() - 1000)
        })
        expect(latest).toEqual(season)

        act(() => {
            vi.advanceTimersByTime(1000)
        })
        expect(latest).toBeUndefined()
    })
})
