// @vitest-environment jsdom
import {ReactElement} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {EndedSeason, HallOfFameBackend} from "../../backends/hallOfFame.ts"
import {FakeHallOfFameBackend, sampleEndedSeason} from "../../backends/fakeHallOfFameBackend.ts"
import {HALL_RETRIES, HALL_RETRY_MS, useHallOfFame} from "./useHallOfFame.ts"

const SAMPLE_ENDED_SEASON = sampleEndedSeason()

let latest: EndedSeason[] | undefined

function Harness({backend, ended}: {backend?: HallOfFameBackend, ended?: number}) {
    latest = useHallOfFame(backend, ended)
    return null
}

const counting = (answers: EndedSeason[][]) => {
    const hallOfFame = vi.fn(async () => answers[Math.min(hallOfFame.mock.calls.length - 1, answers.length - 1)])
    return {hallOfFame}
}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("useHallOfFame", () => {
    it("has no hall with no backend", () => {
        render(<Harness/>)
        expect(latest).toBeUndefined()
    })

    it("holds the seasons the backend answers", async () => {
        await act(async () => {
            render(<Harness backend={new FakeHallOfFameBackend([SAMPLE_ENDED_SEASON])}/>)
        })
        expect(latest).toEqual([SAMPLE_ENDED_SEASON])
    })

    it("reads once when the season the race closed is already in it", async () => {
        const backend = counting([[SAMPLE_ENDED_SEASON]])
        let rerender: (ui: ReactElement) => void = () => {}
        await act(async () => {
            rerender = render(<Harness backend={backend}/>).rerender
        })

        await act(async () => {
            rerender(<Harness backend={backend} ended={SAMPLE_ENDED_SEASON.number}/>)
        })

        expect(backend.hallOfFame).toHaveBeenCalledTimes(1)
    })

    it("reads again when a season ends, until the season is in it", async () => {
        vi.useFakeTimers()
        const backend = counting([[], [], [SAMPLE_ENDED_SEASON]])
        let rerender: (ui: ReactElement) => void = () => {}
        await act(async () => {
            rerender = render(<Harness backend={backend}/>).rerender
        })
        expect(latest).toEqual([])

        await act(async () => {
            rerender(<Harness backend={backend} ended={0}/>)
        })
        expect(backend.hallOfFame).toHaveBeenCalledTimes(2)
        expect(latest).toEqual([])

        await act(async () => {
            await vi.advanceTimersByTimeAsync(HALL_RETRY_MS)
        })
        expect(latest).toEqual([SAMPLE_ENDED_SEASON])

        await act(async () => {
            await vi.advanceTimersByTimeAsync(10 * HALL_RETRY_MS)
        })
        expect(backend.hallOfFame).toHaveBeenCalledTimes(3)
    })

    it("stops reading again after a few tries", async () => {
        vi.useFakeTimers()
        const backend = counting([[]])
        await act(async () => {
            render(<Harness backend={backend} ended={0}/>)
        })

        await act(async () => {
            await vi.advanceTimersByTimeAsync((HALL_RETRIES + 3) * HALL_RETRY_MS)
        })

        expect(backend.hallOfFame).toHaveBeenCalledTimes(1 + HALL_RETRIES)
    })

    it("says nothing when the read fails", async () => {
        const error = vi.spyOn(console, "error").mockImplementation(() => {})
        await act(async () => {
            render(<Harness backend={{hallOfFame: () => Promise.reject(new Error("down"))}}/>)
        })
        expect(latest).toBeUndefined()
        expect(error).toHaveBeenCalled()
    })
})
