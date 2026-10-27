// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {useEffect} from "react"
import {act, renderHook} from "@testing-library/react"
import {ClosedRound, Race} from "../../backends/standings.ts"
import {REVEAL_SEEN_STORAGE_KEY, REVEAL_WITHIN_MS, revealKeyOf} from "../../domain/roundReveal.ts"
import {useAnnouncements} from "../announcements/useAnnouncements.ts"
import {useRoundReveal} from "./useRoundReveal.ts"

const NOW = Date.UTC(2026, 9, 16, 22)

const closedAt = (endedAt: number): ClosedRound => ({
    season: 0, number: 5, endedAt, finale: false, standings: [], before: [], after: [],
    own: false, times: 1, startedAt: 0, takes: [], players: [],
})
const finalBattleAt = (endedAt: number): ClosedRound =>
    ({...closedAt(endedAt), number: 1, finale: true, own: true, times: 3, startedAt: endedAt - 7_200_000})
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

        expect(result.current.due).toEqual([{kind: "round", closed}])
    })

    it("plays a round that closes while the page is open", () => {
        const {result, rerender} = renderHook(({race}) => useRoundReveal(race), {initialProps: {race: raceWith()}})
        expect(result.current.due).toEqual([])

        vi.setSystemTime(NOW + 2 * REVEAL_WITHIN_MS)
        const closed = closedAt(NOW + 2 * REVEAL_WITHIN_MS)
        rerender({race: raceWith(closed)})

        expect(result.current.due).toEqual([{kind: "round", closed}])
    })

    it("does not play a round closed long before the page opened", () => {
        const {result} = renderHook(() => useRoundReveal(raceWith(closedAt(NOW - REVEAL_WITHIN_MS))))

        expect(result.current.due).toEqual([])
    })

    it("plays a round once, across reloads", () => {
        const closed = closedAt(NOW - 60_000)
        const {result} = renderHook(() => useRoundReveal(raceWith(closed)))

        act(() => result.current.dismiss({kind: "round", closed}))

        expect(result.current.due).toEqual([])
        expect(window.localStorage.getItem(REVEAL_SEEN_STORAGE_KEY)).toBe(revealKeyOf({kind: "round", closed}))
        expect(renderHook(() => useRoundReveal(raceWith(closed))).result.current.due).toEqual([])
    })

    it("plays the Final Battle's brief, then the season's results, each once", () => {
        const closed = finalBattleAt(NOW - 60_000)
        const {result} = renderHook(() => useRoundReveal(raceWith(closed)))
        expect(result.current.due).toEqual([{kind: "brief", closed}, {kind: "results", closed}])

        act(() => result.current.dismiss({kind: "brief", closed}))
        expect(result.current.due).toEqual([{kind: "results", closed}])
        expect(renderHook(() => useRoundReveal(raceWith(closed))).result.current.due).toEqual([{kind: "results", closed}])

        act(() => result.current.dismiss({kind: "results", closed}))
        expect(result.current.due).toEqual([])
    })

    it("queues the brief before the season's results: closing the brief shows the results", () => {
        const closed = finalBattleAt(NOW - 60_000)
        const {result} = renderHook(() => {
            const reveal = useRoundReveal(raceWith(closed))
            const {announce, current, dismiss} = useAnnouncements()
            useEffect(() => {
                for (const due of reveal.due) announce(due)
            }, [announce, reveal.due])
            return {current, close: () => {
                if (current?.kind === "brief" || current?.kind === "results") reveal.dismiss(current)
                dismiss()
            }}
        })

        expect(result.current.current?.kind).toBe("brief")
        act(() => result.current.close())
        expect(result.current.current?.kind).toBe("results")
        act(() => result.current.close())
        expect(result.current.current).toBeUndefined()
    })
})
