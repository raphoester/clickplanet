import {describe, expect, it, vi} from "vitest"
import {RateLimitedError} from "../../backends/backend.ts"
import {ALL_OFF} from "../../domain/bonus.ts"
import {acceptedClicks} from "./acceptedClicks.ts"

describe("acceptedClicks", () => {
    it("passes the click on as it was made", async () => {
        const clickTile = vi.fn(async () => undefined)
        const switches = {...ALL_OFF, spread: true}

        await acceptedClicks({clickTile}).clicker.clickTile(42, "fr", switches)

        expect(clickTile).toHaveBeenCalledWith(42, "fr", switches)
    })

    it("tells every listener once the server took the click", async () => {
        const clicks = acceptedClicks({clickTile: async () => undefined})
        const first = vi.fn()
        const second = vi.fn()
        clicks.listenForClicks(first)
        clicks.listenForClicks(second)

        await clicks.clicker.clickTile(1, "fr")

        expect(first).toHaveBeenCalledTimes(1)
        expect(second).toHaveBeenCalledTimes(1)
    })

    it("tells nobody of a refused click, and hands the refusal back", async () => {
        const clicks = acceptedClicks({
            clickTile: async () => {
                throw new RateLimitedError()
            },
        })
        const listener = vi.fn()
        clicks.listenForClicks(listener)

        await expect(clicks.clicker.clickTile(1, "fr")).rejects.toThrow(RateLimitedError)
        expect(listener).not.toHaveBeenCalled()
    })

    it("stops telling a listener that stopped listening", async () => {
        const clicks = acceptedClicks({clickTile: async () => undefined})
        const listener = vi.fn()
        const stop = clicks.listenForClicks(listener)

        stop()
        await clicks.clicker.clickTile(1, "fr")

        expect(listener).not.toHaveBeenCalled()
    })
})
