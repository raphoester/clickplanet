// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {DayMissions, MissionsBackend} from "../../backends/missions.ts"
import {acceptedClicks} from "../clicks/acceptedClicks.ts"
import {SETTLE_MS} from "../clicks/useAcceptedClicks.ts"
import {NO_MISSIONS, useMissions} from "./missionsStore.ts"
import {useMissionsFeed} from "./useMissionsFeed.ts"

const day: DayMissions = {endsAt: Date.UTC(2026, 9, 26, 21, 0), missions: []}

const paid: DayMissions = {...day, missions: [{
    id: "bank", order: {kind: "takeFor", country: "fr", goal: 60, taken: 60}, reward: {kind: "bomb", amount: 1}, done: true, paid: true,
}]}

function followable(missions: MissionsBackend["missions"]) {
    let send: (day: DayMissions) => void = () => {}
    const stop = vi.fn()
    const backend: MissionsBackend = {
        missions: vi.fn(missions),
        follow: vi.fn((onDay: (day: DayMissions) => void) => {
            send = onDay
            return stop
        }),
    }
    return {backend, stop, send: (next: DayMissions) => act(() => send(next))}
}

const state = () => useMissions.getState().state

afterEach(() => {
    cleanup()
    useMissions.setState(NO_MISSIONS)
    vi.useRealTimers()
})

describe("useMissionsFeed", () => {
    it("reads once the clicks settle, and no more", async () => {
        vi.useFakeTimers({now: Date.UTC(2026, 9, 26, 12, 0)})
        const {backend} = followable(async () => day)
        const clicks = acceptedClicks()
        renderHook(() => useMissionsFeed(backend, "acc-1", "fr", clicks.listenForClicks))
        await act(async () => {})
        expect(backend.missions).toHaveBeenCalledTimes(1)

        act(() => clicks.record({country: "fr", took: true}))
        await act(async () => vi.advanceTimersByTime(SETTLE_MS))
        expect(backend.missions).toHaveBeenCalledTimes(2)

        await act(async () => vi.advanceTimersByTime(10 * SETTLE_MS))
        expect(backend.missions).toHaveBeenCalledTimes(2)
    })

    it("shows each day the stream sends, and follows the account again when it changes", async () => {
        const {backend, stop, send} = followable(async () => day)
        const {rerender} = renderHook(({account}) => useMissionsFeed(backend, account, "fr", undefined),
            {initialProps: {account: "acc-1"}})
        await act(async () => {})

        send(paid)
        expect(state()).toEqual({kind: "ready", day: paid})

        rerender({account: "acc-2"})
        expect(stop).toHaveBeenCalledTimes(1)
        expect(backend.follow).toHaveBeenCalledTimes(2)
    })

    it("follows nothing before the account is known", () => {
        const {backend} = followable(async () => day)

        renderHook(() => useMissionsFeed(backend, undefined, "fr", undefined))

        expect(backend.follow).not.toHaveBeenCalled()
    })

    it("keeps a streamed day over a read that started before it", async () => {
        let answer: (day: DayMissions) => void = () => {}
        const {backend, send} = followable(() => new Promise((resolve) => {
            answer = resolve
        }))
        renderHook(() => useMissionsFeed(backend, "acc-1", "fr", undefined))

        send(paid)
        await act(async () => answer(day))

        expect(state()).toEqual({kind: "ready", day: paid})
    })

    it("puts the store back once it stops", async () => {
        const {backend} = followable(async () => day)
        const {unmount} = renderHook(() => useMissionsFeed(backend, "acc-1", "fr", undefined))
        await act(async () => {})
        expect(state()).toEqual({kind: "ready", day})

        unmount()

        expect(state()).toEqual({kind: "loading"})
    })
})
