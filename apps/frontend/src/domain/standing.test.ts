import {describe, expect, it} from "vitest"
import {standingOf, sameStanding, stands} from "./standing.ts"

describe("the flags still in while the knock-out plays", () => {
    const standing = standingOf(new Map([[1, "fr"], [2, "de"], [3, "it"]]), new Set(["de"]))

    it("are the capitals' countries not knocked out", () => {
        expect(stands(standing, "fr")).toBe(true)
        expect(stands(standing, "de")).toBe(false)
    })

    it("leave out a flag with no capital", () => {
        expect(stands(standing, "gb-eng")).toBe(false)
    })

    it("are every flag while the knock-out does not play", () => {
        expect(stands(undefined, "gb-eng")).toBe(true)
    })

    it("are the same when they hold the same flags", () => {
        const again = standingOf(new Map([[3, "it"], [2, "de"], [1, "fr"]]), new Set(["de"]))
        expect(sameStanding(standing, again)).toBe(true)
        expect(sameStanding(standing, standingOf(new Map([[1, "fr"]]), new Set()))).toBe(false)
        expect(sameStanding(standing, {...standing, out: new Set()})).toBe(false)
        expect(sameStanding(undefined, standing)).toBe(false)
        expect(sameStanding(undefined, undefined)).toBe(true)
    })
})
