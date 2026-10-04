// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {NameColor} from "../../backends/player.ts"
import {Standing, StandingsBackend} from "../../backends/standings.ts"
import {useStandings} from "./useStandings.ts"

const ana: Standing = {rank: 1, name: "Ana", color: NameColor.PINK, countryCode: "fr", tiles: 40}
const kofi: Standing = {rank: 1, name: "Kofi", color: NameColor.GREEN, countryCode: "gh", tiles: 12}

let latest: readonly Standing[] | undefined

function Harness({backend, countryCode}: {backend: StandingsBackend, countryCode: string}) {
    latest = useStandings(backend, countryCode)
    return null
}

function streaming() {
    const followers = new Map<string, (standings: Standing[]) => void>()
    const stops = new Map<string, ReturnType<typeof vi.fn>>()
    const backend = {
        listenForStandings: vi.fn((countryCode: string, onStandings: (standings: Standing[]) => void) => {
            followers.set(countryCode, onStandings)
            const stop = vi.fn()
            stops.set(countryCode, stop)
            return stop
        }),
        mySeason: vi.fn(async () => undefined),
    } satisfies StandingsBackend
    const send = (countryCode: string, standings: Standing[]) => act(() => followers.get(countryCode)!(standings))
    return {backend, send, stopOf: (countryCode: string) => stops.get(countryCode)!}
}

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("useStandings", () => {
    it("holds nothing until the first board comes, then each board the stream sends", () => {
        const {backend, send} = streaming()
        render(<Harness backend={backend} countryCode=""/>)
        expect(latest).toBeUndefined()
        expect(backend.listenForStandings).toHaveBeenCalledWith("", expect.any(Function))

        send("", [ana])
        expect(latest).toEqual([ana])

        send("", [kofi, ana])
        expect(latest).toEqual([kofi, ana])
    })

    it("follows another country's players and shows them only once their board comes, never the last country's", () => {
        const {backend, send, stopOf} = streaming()
        const view = render(<Harness backend={backend} countryCode="fr"/>)
        send("fr", [ana])

        view.rerender(<Harness backend={backend} countryCode="gh"/>)
        expect(stopOf("fr")).toHaveBeenCalled()
        expect(latest).toBeUndefined()

        send("gh", [kofi])
        expect(latest).toEqual([kofi])
    })

    it("stops following once it is gone, and takes nothing sent after", () => {
        const {backend, send, stopOf} = streaming()
        const view = render(<Harness backend={backend} countryCode=""/>)
        send("", [ana])

        view.unmount()
        send("", [kofi])

        expect(stopOf("")).toHaveBeenCalled()
        expect(latest).toEqual([ana])
    })
})
