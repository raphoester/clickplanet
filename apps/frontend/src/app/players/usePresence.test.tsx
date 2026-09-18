// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {PresenceBackend} from "../../backends/player.ts"
import {Announcing} from "../../domain/presence.ts"
import {usePresence} from "./usePresence.ts"

function Harness({backend, announcing}: {backend?: PresenceBackend, announcing: Announcing}) {
    usePresence(backend, announcing)
    return null
}

function backendHolding(session: {current: string | undefined}) {
    return {
        heldSession: vi.fn(() => session.current),
        announce: vi.fn(async () => true),
        leave: vi.fn(),
        listenForRoster: vi.fn(() => () => {}),
    }
}

const france: Announcing = {countryCode: "fr"}

const advance = (ms: number) => act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
})

beforeEach(() => vi.useFakeTimers())

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

// The rules are PresenceSchedule's and tested there; this is the wiring.
describe("usePresence", () => {
    it("announces once a click has minted a token, and not before", async () => {
        const session = {current: undefined as string | undefined}
        const backend = backendHolding(session)
        render(<Harness backend={backend} announcing={france}/>)

        await advance(5_000)
        expect(backend.announce).not.toHaveBeenCalled()

        session.current = "token-1"
        await advance(1_000)
        expect(backend.announce).toHaveBeenCalledWith({countryCode: "fr"})
    })

    it("announces the flag the player moves to", async () => {
        const backend = backendHolding({current: "token-1"})
        const view = render(<Harness backend={backend} announcing={france}/>)
        await advance(0)
        expect(backend.announce).toHaveBeenCalledTimes(1)

        view.rerender(<Harness backend={backend} announcing={{...france, countryCode: "jp"}}/>)
        await advance(2_000)

        expect(backend.announce).toHaveBeenCalledTimes(2)
        expect(backend.announce).toHaveBeenLastCalledWith({countryCode: "jp"})
    })

    it("leaves when the page closes, and not when it is only kept for the back button", () => {
        const backend = backendHolding({current: "token-1"})
        render(<Harness backend={backend} announcing={france}/>)

        window.dispatchEvent(new PageTransitionEvent("pagehide", {persisted: true}))
        expect(backend.leave).not.toHaveBeenCalled()

        window.dispatchEvent(new PageTransitionEvent("pagehide", {persisted: false}))
        expect(backend.leave).toHaveBeenCalledTimes(1)
    })

    it("keeps going after an announce that failed", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const backend = backendHolding({current: "token-1"})
        backend.announce.mockRejectedValueOnce(new Error("offline"))
        render(<Harness backend={backend} announcing={france}/>)

        await advance(30_000)

        expect(backend.announce).toHaveBeenCalledTimes(2)
    })
})
