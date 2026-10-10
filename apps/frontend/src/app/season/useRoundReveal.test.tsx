// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, renderHook} from "@testing-library/react"
import {ClosedRound, Race} from "../../backends/standings.ts"
import {REVEAL_SEEN_STORAGE_KEY, REVEAL_WITHIN_MS, revealKeyOf} from "../../domain/roundReveal.ts"
import {useRoundReveal} from "./useRoundReveal.ts"

const NOW = Date.UTC(2026, 9, 16, 22)

const closedAt = (endedAt: number): ClosedRound =>
    ({season: 0, number: 5, endedAt, finale: false, standings: [], before: [], after: []})
const raceWith = (closed?: ClosedRound): Race => ({scores: [], closed})

beforeEach(() => {
    vi.useFakeTimers({now: NOW})
    window.localStorage.clear()
})

afterEach(() => {
    vi.useRealTimers()
})

describe("useRoundReveal", () => {
    it("plays a round closed a little before the page opened", () => {
        const closed = closedAt(NOW - 60 * 60 * 1000)

        const {result} = renderHook(() => useRoundReveal(raceWith(closed)))

        expect(result.current.closed).toEqual(closed)
    })

    it("plays a round that closes while the page is open", () => {
        const {result, rerender} = renderHook(({race}) => useRoundReveal(race), {initialProps: {race: raceWith()}})
        expect(result.current.closed).toBeUndefined()

        vi.setSystemTime(NOW + 2 * REVEAL_WITHIN_MS)
        const closed = closedAt(NOW + 2 * REVEAL_WITHIN_MS)
        rerender({race: raceWith(closed)})

        expect(result.current.closed).toEqual(closed)
    })

    it("does not play a round closed long before the page opened", () => {
        const {result} = renderHook(() => useRoundReveal(raceWith(closedAt(NOW - REVEAL_WITHIN_MS))))

        expect(result.current.closed).toBeUndefined()
    })

    it("plays a round once, across reloads", () => {
        const closed = closedAt(NOW - 60_000)
        const {result} = renderHook(() => useRoundReveal(raceWith(closed)))

        act(() => result.current.dismiss())

        expect(result.current.closed).toBeUndefined()
        expect(window.localStorage.getItem(REVEAL_SEEN_STORAGE_KEY)).toBe(revealKeyOf(closed))
        expect(renderHook(() => useRoundReveal(raceWith(closed))).result.current.closed).toBeUndefined()
    })
})
