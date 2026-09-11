// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {SAMPLE_MS, useLeaderboardFeed} from "./useLeaderboardFeed.ts"
import {Countries} from "../../domain/countries.ts"
import {DELTA_HOLD_MS} from "../../domain/tileDeltas.ts"

const board = (pairs: Record<string, number>) =>
    Object.entries(pairs).map(([code, tiles]) => ({country: Countries.get(code)!, tiles}))

let latest: ReturnType<typeof useLeaderboardFeed>
let renders = 0

function Harness() {
    renders++
    latest = useLeaderboardFeed()
    return null
}

const record = (pairs: Record<string, number>, live = true) =>
    act(() => latest.recordLeaderboard(board(pairs), live))

const tick = (times = 1) => act(() => vi.advanceTimersByTime(SAMPLE_MS * times))

const nets = () => Object.fromEntries([...latest.tileDeltas].map(([c, b]) => [c, b.net]))

const tiles = () => Object.fromEntries(latest.leaderboard.map(e => [e.country.code, e.tiles]))

beforeEach(() => {
    vi.useFakeTimers()
    renders = 0
    render(<Harness/>)
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("useLeaderboardFeed", () => {
    it("holds a board back until the next sample", () => {
        record({fr: 500}, false)
        expect(tiles()).toEqual({})

        tick()
        expect(tiles()).toEqual({fr: 500})
    })

    it("renders once for a burst of boards, not once each", () => {
        record({fr: 500}, false)
        tick()
        const settled = renders

        for (let i = 1; i <= 20; i++) record({fr: 500 + i})
        expect(renders).toBe(settled)

        tick()
        expect(renders).toBe(settled + 1)
        expect(tiles()).toEqual({fr: 520})
    })

    it("gathers a burst into one badge, counted from every board in it", () => {
        record({fr: 500}, false)
        tick()

        record({fr: 501})
        record({fr: 502})
        record({fr: 503})
        tick()

        expect(nets()).toEqual({fr: 3})
    })

    it("badges nothing for the map the globe loads", () => {
        record({fr: 500, jp: 200}, false)
        tick()
        record({fr: 900, jp: 400}, false)
        tick()

        expect(nets()).toEqual({})
        expect(tiles()).toEqual({fr: 900, jp: 400})
    })

    it("reads the first live board against the map that was loaded", () => {
        record({fr: 500}, false)
        tick()
        record({fr: 501})
        tick()

        expect(nets()).toEqual({fr: 1})
    })

    it("keeps a badge across samples, then drops it", () => {
        record({fr: 10}, false)
        tick()
        record({fr: 11})
        tick()
        expect(nets()).toEqual({fr: 1})

        // The badge was stamped one tick before it was published, so it has
        // DELTA_HOLD_MS less that tick left to run.
        tick(DELTA_HOLD_MS / SAMPLE_MS - 2)
        expect(nets()).toEqual({fr: 1})

        tick()
        expect(nets()).toEqual({})
    })

    it("gives a badge its full time again when more tiles land on it", () => {
        record({fr: 10}, false)
        tick()
        record({fr: 11})
        tick(4)

        record({fr: 12})
        tick()
        expect(nets()).toEqual({fr: 2})

        tick(4)
        expect(nets()).toEqual({})
    })

    it("costs no render while the board is still", () => {
        record({fr: 10}, false)
        tick()
        record({fr: 11})
        tick(DELTA_HOLD_MS / SAMPLE_MS + 1)

        const settled = renders
        tick(10)
        expect(renders).toBe(settled)
    })

    it("stops sampling once it goes away", () => {
        cleanup()
        expect(vi.getTimerCount()).toBe(0)
    })
})
