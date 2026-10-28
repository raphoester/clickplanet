// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {useEffect} from "react"
import {act, renderHook} from "@testing-library/react"
import {ClosedRound} from "../../backends/standings.ts"
import {REVEAL_SEEN_STORAGE_KEY, REVEAL_WITHIN_MS, revealKeyOf} from "../../domain/roundReveal.ts"
import {useAnnouncements} from "../announcements/useAnnouncements.ts"
import {useRoundReveal} from "./useRoundReveal.ts"

const NOW = Date.UTC(2026, 9, 16, 22)

const closedAt = (endedAt: number): ClosedRound => ({
    season: 0, number: 5, endedAt, finale: false, standings: [], before: [], after: [],
    moment: "day", roundId: "season-0/day-5", times: 1, startedAt: 0, entryId: "", takes: [], players: [],
})
const finalBattleAt = (endedAt: number): ClosedRound => ({
    ...closedAt(endedAt), number: 1, finale: true, moment: "battle", times: 3, startedAt: endedAt - 7_200_000, entryId: "entry-final", roundId: "entry-final",
    standings: [{rank: 1, countryCode: "de", share: 0.3, points: 75}],
})

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

        const {result} = renderHook(() => useRoundReveal(closed, undefined))

        expect(result.current.due).toEqual([{kind: "round", closed}])
    })

    it("plays a round that closes while the page is open", () => {
        const {result, rerender} = renderHook(({race}) => useRoundReveal(race, undefined), {initialProps: {race: undefined as ClosedRound | undefined}})
        expect(result.current.due).toEqual([])

        vi.setSystemTime(NOW + 2 * REVEAL_WITHIN_MS)
        const closed = closedAt(NOW + 2 * REVEAL_WITHIN_MS)
        rerender({race: closed})

        expect(result.current.due).toEqual([{kind: "round", closed}])
    })

    it("does not play a round closed long before the page opened", () => {
        const {result} = renderHook(() => useRoundReveal(closedAt(NOW - REVEAL_WITHIN_MS), undefined))

        expect(result.current.due).toEqual([])
    })

    it("plays a round once, across reloads", () => {
        const closed = closedAt(NOW - 60_000)
        const {result} = renderHook(() => useRoundReveal(closed, undefined))

        act(() => result.current.dismiss({kind: "round", closed}))

        expect(result.current.due).toEqual([])
        expect(window.localStorage.getItem(REVEAL_SEEN_STORAGE_KEY)).toBe(revealKeyOf({kind: "round", closed}))
        expect(renderHook(() => useRoundReveal(closed, undefined)).result.current.due).toEqual([])
    })

    it("plays the Final Battle's debrief, then the season's results, each once", () => {
        const closed = finalBattleAt(NOW - 60_000)
        const {result} = renderHook(() => useRoundReveal(closed, undefined))
        expect(result.current.due).toEqual([{kind: "over", closed}, {kind: "points", closed}, {kind: "results", closed}])

        act(() => result.current.dismiss({kind: "points", closed}))
        expect(result.current.due).toEqual([{kind: "results", closed}])
        expect(renderHook(() => useRoundReveal(closed, undefined)).result.current.due).toEqual([{kind: "results", closed}])

        act(() => result.current.dismiss({kind: "results", closed}))
        expect(result.current.due).toEqual([])
    })

    it("plays the debrief of the battle the page saw by its round, and not another battle's", () => {
        const closed = finalBattleAt(NOW - 60_000)
        const battle = {entryId: "entry-final", since: closed.startedAt, endsAt: closed.endedAt, description: "", roundId: "entry-final"}

        expect(renderHook(() => useRoundReveal(closed, battle)).result.current.due.map(({kind}) => kind))
            .toEqual(["over", "points", "results"])
        expect(renderHook(() => useRoundReveal(closed, {...battle, entryId: "entry-before", roundId: "entry-before"})).result.current.due)
            .toEqual([{kind: "results", closed}])
    })

    it("queues the debrief's screens before the season's results, in order", () => {
        const closed = finalBattleAt(NOW - 60_000)
        const {result} = renderHook(() => {
            const reveal = useRoundReveal(closed, undefined)
            const {announce, current, dismiss} = useAnnouncements()
            useEffect(() => {
                for (const due of reveal.due) announce(due)
            }, [announce, reveal.due])
            return {current, close: () => {
                if (current?.kind === "over" || current?.kind === "points" || current?.kind === "results") reveal.dismiss(current)
                dismiss()
            }}
        })

        expect(result.current.current?.kind).toBe("over")
        act(() => result.current.close())
        expect(result.current.current?.kind).toBe("points")
        act(() => result.current.close())
        expect(result.current.current?.kind).toBe("results")
        act(() => result.current.close())
        expect(result.current.current).toBeUndefined()
    })
})
