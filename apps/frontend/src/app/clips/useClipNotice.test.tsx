// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {StrictMode} from "react"
import {Appearance, ClipsBackend} from "../../backends/clips.ts"
import {APPEARANCES_READ_GAP_MS, CLIPS_SHOWN_STORAGE_KEY} from "../../domain/clipAppearances.ts"
import {ClipNotice, useClipNotice} from "./useClipNotice.ts"

const clip = (postId: string, scheduledAt: number): Appearance =>
    ({postId, title: "FRANCE VS TURKEY", scheduledAt, links: [{platform: "tiktok", url: ""}]})

let latest: ClipNotice

function Harness({backend, account}: {backend: ClipsBackend, account?: string}) {
    latest = useClipNotice(backend, account)
    return null
}

function backendWith(...reads: Appearance[][]) {
    const appearances = vi.fn(async () => reads[Math.min(appearances.mock.calls.length - 1, reads.length - 1)])
    return {appearances}
}

function setHidden(hidden: boolean) {
    Object.defineProperty(document, "hidden", {configurable: true, get: () => hidden})
    act(() => {
        document.dispatchEvent(new Event("visibilitychange"))
    })
}

beforeEach(() => {
    window.localStorage.clear()
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    setHidden(false)
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("useClipNotice", () => {
    it("reads the clips of a named player and shows the newest, once per browser", async () => {
        render(<Harness backend={backendWith([clip("post-1", 1), clip("post-2", 2)])} account="account-1"/>)

        await vi.waitFor(() => expect(latest.appearance?.postId).toBe("post-2"))
        act(() => latest.dismiss(latest.appearance!))

        expect(latest.appearance).toBeUndefined()
        expect(JSON.parse(window.localStorage.getItem(CLIPS_SHOWN_STORAGE_KEY)!)).toEqual(["post-2", "post-1"])

        cleanup()
        const reloaded = backendWith([clip("post-1", 1), clip("post-2", 2)])
        render(<Harness backend={reloaded} account="account-1"/>)
        await act(async () => {})

        expect(reloaded.appearances).toHaveBeenCalled()
        expect(latest.appearance).toBeUndefined()
    })

    it("reads nothing for a guest", async () => {
        const backend = backendWith([clip("post-1", 1)])
        render(<Harness backend={backend}/>)
        await act(async () => {})

        expect(backend.appearances).not.toHaveBeenCalled()
        expect(latest.appearance).toBeUndefined()
    })

    it("reads once under StrictMode, and still shows what it read", async () => {
        const backend = backendWith([clip("post-1", 1)])
        render(<StrictMode><Harness backend={backend} account="account-1"/></StrictMode>)

        await vi.waitFor(() => expect(latest.appearance?.postId).toBe("post-1"))
        expect(backend.appearances).toHaveBeenCalledTimes(1)
    })

    it("reads again when the tab comes back, at most once every ten minutes", async () => {
        const backend = backendWith([], [clip("post-1", 1)])
        render(<Harness backend={backend} account="account-1"/>)
        await act(async () => {})

        setHidden(true)
        setHidden(false)
        expect(backend.appearances).toHaveBeenCalledTimes(1)

        act(() => vi.advanceTimersByTime(APPEARANCES_READ_GAP_MS))
        setHidden(true)
        setHidden(false)

        await vi.waitFor(() => expect(latest.appearance?.postId).toBe("post-1"))
        expect(backend.appearances).toHaveBeenCalledTimes(2)
    })

    it("reads at once for a player who signs in again", async () => {
        const backend = backendWith([], [clip("post-1", 1)])
        const {rerender} = render(<Harness backend={backend} account="account-1"/>)
        await act(async () => {})

        rerender(<Harness backend={backend}/>)
        rerender(<Harness backend={backend} account="account-1"/>)

        await vi.waitFor(() => expect(latest.appearance?.postId).toBe("post-1"))
    })

    it("never shows another account's clip", async () => {
        const backend = backendWith([clip("post-1", 1)], [])
        const {rerender} = render(<Harness backend={backend} account="account-1"/>)
        await vi.waitFor(() => expect(latest.appearance?.postId).toBe("post-1"))

        rerender(<Harness backend={backend} account="account-2"/>)

        expect(latest.appearance).toBeUndefined()
    })

    it("works without storage", async () => {
        vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
            throw new Error("blocked")
        })
        vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
            throw new Error("blocked")
        })
        vi.spyOn(console, "error").mockImplementation(() => {})
        render(<Harness backend={backendWith([clip("post-1", 1)])} account="account-1"/>)
        await vi.waitFor(() => expect(latest.appearance?.postId).toBe("post-1"))

        act(() => latest.dismiss(latest.appearance!))

        expect(latest.appearance).toBeUndefined()
    })
})
