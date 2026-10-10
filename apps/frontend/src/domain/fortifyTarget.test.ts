import {describe, expect, it} from "vitest"
import {CLOSE_TILES, targetOf} from "./fortifyTarget.ts"

describe("targetOf", () => {
    it("names the leader and how many tiles it misses", () => {
        expect(targetOf(2834, "us", 2812, undefined)).toEqual({flag: "us", missing: 22})
        expect(targetOf(2834, "us", 2812, "ca")).toEqual({flag: "us", missing: 22})
    })

    it("says nothing to a leader that fortified it last", () => {
        expect(targetOf(16, "bg", 15, "bg")).toBeUndefined()
    })

    it("says nothing to a leader too far from the whole", () => {
        expect(targetOf(2834, "us", 2834 - CLOSE_TILES - 1, undefined)).toBeUndefined()
    })

    it("says nothing to a leader that does not hold most of it", () => {
        expect(targetOf(30, "fr", 15, undefined)).toBeUndefined()
        expect(targetOf(2, "fr", 1, undefined)).toBeUndefined()
    })

    it("says nothing once the landmass is whole, or nobody holds it", () => {
        expect(targetOf(16, "bg", 16, undefined)).toBeUndefined()
        expect(targetOf(16, undefined, 0, undefined)).toBeUndefined()
    })
})
