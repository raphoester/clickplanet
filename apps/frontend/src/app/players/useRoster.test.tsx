// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {PresenceBackend, RosterEntry, RosterUnavailableError} from "../../backends/player.ts"
import {ROSTER_EVERY_MS, RosterState, useRoster} from "./useRoster.ts"

const ana: RosterEntry = {name: "ana", tag: "4f2ca1", countryCode: "fr", guest: false}
const bo: RosterEntry = {name: "guest_Bo", tag: "91aa3d", countryCode: "de", guest: true}

let latest: RosterState
let visibility: DocumentVisibilityState = "visible"

function Harness({backend}: {backend?: PresenceBackend}) {
    latest = useRoster(backend)
    return null
}

function backendAnswering(roster: PresenceBackend["roster"]) {
    return {heldSession: vi.fn(), announce: vi.fn(), roster: vi.fn(roster)}
}

/** Lets the promises a tick started settle, under fake timers. */
const settle = () => act(async () => {
    await vi.advanceTimersByTimeAsync(0)
})

const advance = (ms: number) => act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
})

function setVisibility(next: DocumentVisibilityState) {
    visibility = next
    document.dispatchEvent(new Event("visibilitychange"))
}

beforeEach(() => {
    vi.useFakeTimers()
    visibility = "visible"
    vi.spyOn(document, "visibilityState", "get").mockImplementation(() => visibility)
    vi.spyOn(console, "error").mockImplementation(() => {})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("useRoster", () => {
    it("is unavailable with no backend", () => {
        render(<Harness/>)
        expect(latest).toEqual({kind: "unavailable"})
    })

    it("reads at once, then every ten seconds", async () => {
        const backend = backendAnswering(async () => [ana])
        render(<Harness backend={backend}/>)
        expect(latest).toEqual({kind: "loading"})

        await settle()
        expect(latest).toEqual({kind: "ready", entries: [ana]})
        expect(backend.roster).toHaveBeenCalledTimes(1)

        backend.roster.mockResolvedValue([ana, bo])
        await advance(ROSTER_EVERY_MS)

        expect(backend.roster).toHaveBeenCalledTimes(2)
        expect(latest).toEqual({kind: "ready", entries: [ana, bo]})
    })

    it("asks nothing while the tab is hidden, and asks at once when it is back", async () => {
        const backend = backendAnswering(async () => [ana])
        render(<Harness backend={backend}/>)
        await settle()

        act(() => setVisibility("hidden"))
        await advance(3 * ROSTER_EVERY_MS)
        expect(backend.roster).toHaveBeenCalledTimes(1)

        act(() => setVisibility("visible"))
        await settle()
        expect(backend.roster).toHaveBeenCalledTimes(2)
    })

    it("keeps the last list when a read fails", async () => {
        const backend = backendAnswering(async () => [ana])
        render(<Harness backend={backend}/>)
        await settle()

        backend.roster.mockRejectedValue(new Error("offline"))
        await advance(ROSTER_EVERY_MS)

        expect(latest).toEqual({kind: "ready", entries: [ana]})
    })

    it("hides the list for good on a server without a roster", async () => {
        const backend = backendAnswering(async () => {
            throw new RosterUnavailableError()
        })
        render(<Harness backend={backend}/>)
        await settle()

        expect(latest).toEqual({kind: "unavailable"})

        await advance(3 * ROSTER_EVERY_MS)
        act(() => setVisibility("visible"))
        await settle()
        expect(backend.roster).toHaveBeenCalledTimes(1)
    })
})
