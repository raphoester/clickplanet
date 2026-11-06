import {describe, expect, it, vi} from "vitest"
import {acceptedClicks} from "./acceptedClicks.ts"

describe("acceptedClicks", () => {
    it("tells every listener of an accepted click, and whether it took the tile", () => {
        const clicks = acceptedClicks()
        const first = vi.fn()
        const second = vi.fn()
        clicks.listenForClicks(first)
        clicks.listenForClicks(second)

        clicks.record({country: "fr", took: true})

        expect(first).toHaveBeenCalledWith({country: "fr", took: true})
        expect(second).toHaveBeenCalledWith({country: "fr", took: true})
    })

    it("stops telling a listener that stopped listening", () => {
        const clicks = acceptedClicks()
        const listener = vi.fn()
        const stop = clicks.listenForClicks(listener)

        stop()
        clicks.record({country: "fr", took: false})

        expect(listener).not.toHaveBeenCalled()
    })
})
