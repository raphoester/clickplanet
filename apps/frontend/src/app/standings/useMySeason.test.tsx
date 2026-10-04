// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {MySeason, StandingsBackend} from "../../backends/standings.ts"
import {acceptedClicks} from "../viewer/acceptedClicks.ts"
import {AFTER_CLICKS_MS, Caller, useMySeason} from "./useMySeason.ts"

const GUEST: Caller = {linked: false}

let latest: MySeason | undefined

function Harness({backend, caller, clicks}: {backend: StandingsBackend, caller: Caller, clicks: ReturnType<typeof acceptedClicks>}) {
    latest = useMySeason(backend, caller, clicks.listenForClicks)
    return null
}

function counting() {
    let tiles = 0
    return {
        standings: vi.fn(async () => []),
        mySeason: vi.fn(async (): Promise<MySeason> => ({countryCode: "fr", tiles: tiles++})),
    }
}

const clicking = () => acceptedClicks({clickTile: async () => undefined})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("useMySeason", () => {
    it("reads the caller's season once it is shown", async () => {
        const backend = counting()
        render(<Harness backend={backend} caller={GUEST} clicks={clicking()}/>)

        await act(async () => {})

        expect(latest).toEqual({countryCode: "fr", tiles: 0})
        expect(backend.mySeason).toHaveBeenCalledTimes(1)
    })

    it("reads again a few seconds after the last of a run of clicks, once", async () => {
        vi.useFakeTimers()
        const backend = counting()
        const clicks = clicking()
        render(<Harness backend={backend} caller={GUEST} clicks={clicks}/>)
        await act(async () => {})

        await act(async () => {
            await clicks.clicker.clickTile(1, "fr")
            vi.advanceTimersByTime(AFTER_CLICKS_MS - 1)
            await clicks.clicker.clickTile(2, "fr")
            vi.advanceTimersByTime(AFTER_CLICKS_MS - 1)
        })
        expect(backend.mySeason).toHaveBeenCalledTimes(1)

        await act(async () => vi.advanceTimersByTime(1))

        expect(backend.mySeason).toHaveBeenCalledTimes(2)
        expect(latest).toEqual({countryCode: "fr", tiles: 1})
    })

    it("does not read again for a refused click", async () => {
        vi.useFakeTimers()
        const backend = counting()
        const clicks = acceptedClicks({
            clickTile: async () => {
                throw new Error("refused")
            },
        })
        render(<Harness backend={backend} caller={GUEST} clicks={clicks}/>)
        await act(async () => {})

        await act(async () => {
            await clicks.clicker.clickTile(1, "fr").catch(() => undefined)
            vi.advanceTimersByTime(AFTER_CLICKS_MS)
        })

        expect(backend.mySeason).toHaveBeenCalledTimes(1)
    })

    it("reads again when the caller signs in, and when its username changes", async () => {
        const backend = counting()
        const clicks = clicking()
        const view = render(<Harness backend={backend} caller={GUEST} clicks={clicks}/>)
        await act(async () => {})

        view.rerender(<Harness backend={backend} caller={{linked: true}} clicks={clicks}/>)
        await act(async () => {})
        expect(backend.mySeason).toHaveBeenCalledTimes(2)

        view.rerender(<Harness backend={backend} caller={{linked: true, username: "Ana"}} clicks={clicks}/>)
        await act(async () => {})
        expect(backend.mySeason).toHaveBeenCalledTimes(3)

        view.rerender(<Harness backend={backend} caller={{linked: true, username: "Ana"}} clicks={clicks}/>)
        await act(async () => {})
        expect(backend.mySeason).toHaveBeenCalledTimes(3)
        expect(latest).toEqual({countryCode: "fr", tiles: 2})
    })

    it("stops listening for clicks once it is gone", async () => {
        vi.useFakeTimers()
        const backend = counting()
        const clicks = clicking()
        const view = render(<Harness backend={backend} caller={GUEST} clicks={clicks}/>)
        await act(async () => {})

        await act(async () => {
            await clicks.clicker.clickTile(1, "fr")
        })
        view.unmount()
        await act(async () => {
            await clicks.clicker.clickTile(2, "fr")
            vi.advanceTimersByTime(AFTER_CLICKS_MS * 2)
        })

        expect(backend.mySeason).toHaveBeenCalledTimes(1)
    })
})
