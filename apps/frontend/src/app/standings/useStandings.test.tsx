// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {NameColor} from "../../backends/player.ts"
import {Standing, StandingsBackend} from "../../backends/standings.ts"
import {STANDINGS_EVERY_MS, useStandings} from "./useStandings.ts"

const ana: Standing = {rank: 1, name: "Ana", color: NameColor.PINK, countryCode: "fr", tiles: 40}
const kofi: Standing = {rank: 1, name: "Kofi", color: NameColor.GREEN, countryCode: "gh", tiles: 12}

let latest: readonly Standing[] | undefined

function Harness({backend, countryCode}: {backend: StandingsBackend, countryCode: string}) {
    latest = useStandings(backend, countryCode)
    return null
}

function answering(byCountry: Record<string, Standing[]>) {
    return {
        standings: vi.fn(async (countryCode: string) => byCountry[countryCode] ?? []),
        mySeason: vi.fn(async () => undefined),
    }
}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("useStandings", () => {
    it("holds nothing until the first read lands, then the standings", async () => {
        const backend = answering({"": [ana]})
        render(<Harness backend={backend} countryCode=""/>)
        expect(latest).toBeUndefined()

        await act(async () => {})

        expect(latest).toEqual([ana])
        expect(backend.standings).toHaveBeenCalledWith("")
    })

    it("reads again every 15 seconds while it is shown, and stops once it is gone", async () => {
        vi.useFakeTimers()
        const backend = answering({"": [ana]})
        const view = render(<Harness backend={backend} countryCode=""/>)
        await act(async () => {})

        await act(async () => vi.advanceTimersByTime(STANDINGS_EVERY_MS - 1))
        expect(backend.standings).toHaveBeenCalledTimes(1)

        await act(async () => vi.advanceTimersByTime(1))
        expect(backend.standings).toHaveBeenCalledTimes(2)

        view.unmount()
        await act(async () => vi.advanceTimersByTime(STANDINGS_EVERY_MS * 3))
        expect(backend.standings).toHaveBeenCalledTimes(2)
    })

    it("shows another country's players only once they are read, never the last country's", async () => {
        const backend = answering({"fr": [ana], "gh": [kofi]})
        const view = render(<Harness backend={backend} countryCode="fr"/>)
        await act(async () => {})
        expect(latest).toEqual([ana])

        view.rerender(<Harness backend={backend} countryCode="gh"/>)
        expect(latest).toBeUndefined()

        await act(async () => {})
        expect(latest).toEqual([kofi])
    })

    it("keeps the standings it has when a read fails", async () => {
        vi.useFakeTimers()
        const error = vi.spyOn(console, "error").mockImplementation(() => {})
        const backend = answering({"": [ana]})
        render(<Harness backend={backend} countryCode=""/>)
        await act(async () => {})

        backend.standings.mockRejectedValueOnce(new Error("down"))
        await act(async () => vi.advanceTimersByTime(STANDINGS_EVERY_MS))

        expect(latest).toEqual([ana])
        expect(error).toHaveBeenCalled()
    })
})
