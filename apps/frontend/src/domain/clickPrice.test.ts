import {describe, expect, it} from "vitest"
import {describePrice, factor, percent} from "./clickPrice.ts"

describe("describePrice", () => {
    it("says nothing against a server that does not price clicks", () => {
        expect(describePrice(undefined, "Bulgaria")).toBeUndefined()
    })

    it("says nothing for a small country far from the first step", () => {
        expect(describePrice({slowdown: 1, share: 0.02, next: {share: 0.1, slowdown: 2}}, "Chad")).toBeUndefined()
    })

    it("warns a country close to the first step", () => {
        expect(describePrice({slowdown: 1, share: 0.085, next: {share: 0.1, slowdown: 2}}, "France")).toEqual({
            headline: "France holds 8.5% of the map",
            detail: "Refills 2× slower at 10%",
        })
    })

    it("says why a big country refills slowly, and what comes next", () => {
        expect(describePrice({slowdown: 1.5, share: 0.6, next: {share: 0.7, slowdown: 2}}, "Bulgaria")).toEqual({
            headline: "Bulgaria holds 60% of the map",
            detail: "Refills 1.5× slower · 2× at 70%",
        })
    })

    it("names no next step at the top", () => {
        expect(describePrice({slowdown: 2, share: 0.95}, "Bulgaria")?.detail).toBe("Refills 2× slower")
    })
})

describe("factor", () => {
    it("writes a factor the way the config does", () => {
        expect(factor(2)).toBe("2")
        expect(factor(1.5)).toBe("1.5")
        expect(factor(1.25)).toBe("1.25")
    })
})

describe("percent", () => {
    it("rounds down, so a step is never claimed early", () => {
        expect(percent(0.0999)).toBe("9.9%")
        expect(percent(0.2499)).toBe("24%")
        expect(percent(0.8)).toBe("80%")
    })
})
