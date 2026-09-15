import {describe, expect, it} from "vitest"
import {OwnClicks} from "./ownClicks.ts"

describe("OwnClicks", () => {
    it("knows a tile clicked for a country within the window", () => {
        const clicks = new OwnClicks(3)
        clicks.record(7, "fr", 10)
        expect(clicks.has(7, "fr", 12)).toBe(true)
        expect(clicks.has(7, "fr", 13.5)).toBe(false)
    })

    it("does not take another tile or another country for it", () => {
        const clicks = new OwnClicks(3)
        clicks.record(7, "fr", 10)
        expect(clicks.has(8, "fr", 10)).toBe(false)
        expect(clicks.has(7, "de", 10)).toBe(false)
    })

    it("answers every broadcast for the tile, not only the first", () => {
        const clicks = new OwnClicks(3)
        clicks.record(7, "fr", 10)
        expect(clicks.has(7, "fr", 10.1)).toBe(true)
        expect(clicks.has(7, "fr", 10.2)).toBe(true)
    })

    it("drops stale clicks past its limit and keeps fresh ones", () => {
        const clicks = new OwnClicks(3, 2)
        clicks.record(1, "fr", 0)
        clicks.record(2, "fr", 0)
        clicks.record(3, "fr", 10)
        expect(clicks.has(3, "fr", 10)).toBe(true)
        expect(clicks.has(1, "fr", 0)).toBe(false)
    })
})
