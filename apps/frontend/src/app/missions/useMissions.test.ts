// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {DayMissions, MissionsBackend} from "../../backends/missions.ts"
import {acceptedClicks} from "../viewer/acceptedClicks.ts"
import {SETTLE_MS} from "../viewer/useAcceptedClicks.ts"
import {FOLLOW_UP_MS, useMissions} from "./useMissions.ts"

const day: DayMissions = {countryCode: "fr", enlisted: true, endsAt: Date.UTC(2026, 9, 26, 21, 0), missions: []}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("useMissions", () => {
    it("reads once the clicks settle, and once more when the backend has counted them", async () => {
        vi.useFakeTimers({now: Date.UTC(2026, 9, 26, 12, 0)})
        const backend: MissionsBackend = {missions: vi.fn(async () => day)}
        const clicks = acceptedClicks()
        renderHook(() => useMissions(backend, "fr", clicks.listenForClicks))
        await act(async () => {})
        expect(backend.missions).toHaveBeenCalledTimes(1)

        act(() => clicks.record({country: "fr", took: true}))
        await act(async () => vi.advanceTimersByTime(SETTLE_MS))
        expect(backend.missions).toHaveBeenCalledTimes(2)

        await act(async () => vi.advanceTimersByTime(FOLLOW_UP_MS))
        expect(backend.missions).toHaveBeenCalledTimes(3)

        await act(async () => vi.advanceTimersByTime(10 * FOLLOW_UP_MS))
        expect(backend.missions).toHaveBeenCalledTimes(3)
    })
})
