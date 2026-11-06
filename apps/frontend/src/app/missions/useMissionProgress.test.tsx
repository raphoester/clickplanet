// @vitest-environment jsdom
import {StrictMode} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, renderHook} from "@testing-library/react"
import {DayMissions, Mission, MissionOrder} from "../../backends/missions.ts"
import {PROGRESS_LINE_MS, useMissionProgress} from "./useMissionProgress.ts"
import {MissionsState} from "./missionsStore.ts"

function mission(id: string, order: MissionOrder, done = false): Mission {
    return {id, order, reward: {kind: "bomb", amount: 1}, done, paid: done}
}

const rival = (taken: number, done = false) => mission("rival", {kind: "takeFrom", country: "ie", goal: 80, taken}, done)
const ground = (taken: number) => mission("ground", {kind: "takeIn", country: "kz", goal: 300, taken})

function ready(...missions: Mission[]): MissionsState {
    const day: DayMissions = {endsAt: 0, missions}
    return {kind: "ready", day}
}

function follow(off = false) {
    return renderHook(({state}) => useMissionProgress(state, off), {
        initialProps: {state: ready(rival(3), ground(10))},
        wrapper: StrictMode,
    })
}

afterEach(() => {
    vi.useRealTimers()
})

describe("useMissionProgress", () => {
    it("says nothing at the first day of the page", () => {
        expect(follow().result.current).toBeUndefined()
    })

    it("shows the mission that moved, and keeps it while it moves", () => {
        vi.useFakeTimers()
        const hook = follow()

        hook.rerender({state: ready(rival(4), ground(10))})
        expect(hook.result.current?.id).toBe("rival")

        act(() => vi.advanceTimersByTime(PROGRESS_LINE_MS - 500))
        hook.rerender({state: ready(rival(5), ground(11))})
        expect(hook.result.current?.id).toBe("rival")

        act(() => vi.advanceTimersByTime(PROGRESS_LINE_MS - 500))
        expect(hook.result.current?.id).toBe("rival")
    })

    it("lets another mission take the line only once it has gone", () => {
        vi.useFakeTimers()
        const hook = follow()
        hook.rerender({state: ready(rival(4), ground(10))})

        hook.rerender({state: ready(rival(4), ground(11))})
        expect(hook.result.current?.id).toBe("rival")

        act(() => vi.advanceTimersByTime(PROGRESS_LINE_MS))
        expect(hook.result.current).toBeUndefined()

        hook.rerender({state: ready(rival(4), ground(12))})
        expect(hook.result.current?.id).toBe("ground")
    })

    it("goes as soon as its mission is done: the card says it", () => {
        const hook = follow()
        hook.rerender({state: ready(rival(79), ground(10))})

        hook.rerender({state: ready(rival(80, true), ground(10))})

        expect(hook.result.current).toBeUndefined()
    })

    it("shows nothing on a first visit", () => {
        const hook = follow(true)

        hook.rerender({state: ready(rival(4), ground(10))})

        expect(hook.result.current).toBeUndefined()
    })
})
