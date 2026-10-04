// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {MySeason, StandingsBackend} from "../../backends/standings.ts"
import {AcceptedClicks, acceptedClicks} from "../viewer/acceptedClicks.ts"
import {LONGEST_MS, SETTLE_MS} from "../viewer/useAcceptedClicks.ts"
import {Caller, useMySeason} from "./useMySeason.ts"

const GUEST: Caller = {linked: false}

let latest: MySeason | undefined

type HarnessProps = {backend: StandingsBackend, caller: Caller, clicks: AcceptedClicks, countryCode?: string}

function Harness({backend, caller, clicks, countryCode = ""}: HarnessProps) {
    latest = useMySeason(backend, caller, clicks.listenForClicks, countryCode)
    return null
}

function server(tiles = 10) {
    const kept = {tiles, de: 3}
    return {
        kept,
        standings: vi.fn(async () => []),
        mySeason: vi.fn(async (countryCode: string): Promise<MySeason> => countryCode === "de"
            ? {countryCode: "de", tiles: kept.de}
            : {countryCode: "fr", tiles: kept.tiles}),
    }
}

async function shown(backend: StandingsBackend, caller = GUEST, countryCode = "") {
    const clicks = acceptedClicks()
    const view = render(<Harness backend={backend} caller={caller} clicks={clicks} countryCode={countryCode}/>)
    await act(async () => {})
    return {...view, clicks}
}

const take = (clicks: AcceptedClicks, country = "fr") => act(() => clicks.record({country, took: true}))

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("useMySeason", () => {
    it("reads the caller's season once it is shown", async () => {
        const backend = server()
        await shown(backend)

        expect(latest).toEqual({countryCode: "fr", tiles: 10})
        expect(backend.mySeason).toHaveBeenCalledTimes(1)
        expect(backend.mySeason).toHaveBeenCalledWith("")
    })

    it("reads the caller's line on the board of the country shown, and again when that country changes", async () => {
        const backend = server()
        const {clicks, rerender} = await shown(backend, GUEST, "de")
        expect(backend.mySeason).toHaveBeenLastCalledWith("de")
        expect(latest).toEqual({countryCode: "de", tiles: 3})

        let answer: (season: MySeason) => void = () => {}
        backend.mySeason.mockImplementationOnce(() => new Promise((resolve) => answer = resolve))
        rerender(<Harness backend={backend} caller={GUEST} clicks={clicks} countryCode="fr"/>)
        await act(async () => {})

        expect(backend.mySeason).toHaveBeenLastCalledWith("fr")
        expect(latest).toBeUndefined()
        await act(async () => answer({countryCode: "fr", tiles: 4}))
        expect(latest).toEqual({countryCode: "fr", tiles: 4})
    })

    it("counts a take for the country shown into its line there, whatever the main flag", async () => {
        const backend = server()
        const {clicks} = await shown(backend, GUEST, "de")

        await take(clicks, "de")
        await take(clicks, "fr")

        expect(latest).toEqual({countryCode: "de", tiles: 4})
    })

    it("counts each tile the caller takes for its main flag at once", async () => {
        const backend = server()
        const {clicks} = await shown(backend)

        await take(clicks)
        await take(clicks)

        expect(latest).toEqual({countryCode: "fr", tiles: 12})
        expect(backend.mySeason).toHaveBeenCalledTimes(1)
    })

    it("counts nothing for a click that took nothing, or a take for another flag", async () => {
        const backend = server()
        const {clicks} = await shown(backend)

        await act(() => clicks.record({country: "fr", took: false}))
        await take(clicks, "de")

        expect(latest).toEqual({countryCode: "fr", tiles: 10})
    })

    it("reads again a little after the last of a run of clicks, once", async () => {
        vi.useFakeTimers()
        const backend = server()
        const {clicks} = await shown(backend)

        await take(clicks)
        await act(async () => vi.advanceTimersByTime(SETTLE_MS - 1))
        await take(clicks)
        await act(async () => vi.advanceTimersByTime(SETTLE_MS - 1))
        expect(backend.mySeason).toHaveBeenCalledTimes(1)

        backend.kept.tiles = 13
        await act(async () => vi.advanceTimersByTime(1))

        expect(backend.mySeason).toHaveBeenCalledTimes(2)
        expect(latest).toEqual({countryCode: "fr", tiles: 13})
    })

    it("reads at least every few seconds while the clicks keep coming", async () => {
        vi.useFakeTimers()
        const backend = server()
        const {clicks} = await shown(backend)

        for (let elapsed = 0; elapsed < LONGEST_MS; elapsed += SETTLE_MS / 2) {
            await take(clicks)
            await act(async () => vi.advanceTimersByTime(SETTLE_MS / 2))
        }

        expect(backend.mySeason).toHaveBeenCalledTimes(2)
    })

    it("adds the takes made while a read was out to what the read says", async () => {
        vi.useFakeTimers()
        const backend = server()
        const {clicks} = await shown(backend)
        let answer: (season: MySeason) => void = () => {}
        backend.mySeason.mockImplementationOnce(() => new Promise((resolve) => answer = resolve))

        await take(clicks)
        await act(async () => vi.advanceTimersByTime(SETTLE_MS))
        await take(clicks)
        await act(async () => answer({countryCode: "fr", tiles: 11}))

        expect(latest).toEqual({countryCode: "fr", tiles: 12})
    })

    it("reads again when the caller signs in, and when its username changes", async () => {
        const backend = server()
        const {clicks, rerender} = await shown(backend)

        rerender(<Harness backend={backend} caller={{linked: true}} clicks={clicks}/>)
        await act(async () => {})
        expect(backend.mySeason).toHaveBeenCalledTimes(2)

        rerender(<Harness backend={backend} caller={{linked: true, username: "Ana"}} clicks={clicks}/>)
        await act(async () => {})
        expect(backend.mySeason).toHaveBeenCalledTimes(3)

        rerender(<Harness backend={backend} caller={{linked: true, username: "Ana"}} clicks={clicks}/>)
        await act(async () => {})
        expect(backend.mySeason).toHaveBeenCalledTimes(3)
    })

    it("stops listening for clicks once it is gone", async () => {
        vi.useFakeTimers()
        const backend = server()
        const {clicks, unmount} = await shown(backend)

        await take(clicks)
        unmount()
        await act(async () => {
            clicks.record({country: "fr", took: true})
            vi.advanceTimersByTime(LONGEST_MS)
        })

        expect(backend.mySeason).toHaveBeenCalledTimes(1)
    })
})
