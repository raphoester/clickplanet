import {describe, expect, it} from "vitest"
import {factor, priceNote, shareShown} from "./clickPrice.ts"

describe("priceNote", () => {
    it("says nothing against a server that does not price clicks", () => {
        expect(priceNote(undefined)).toBeUndefined()
    })

    it("says nothing for a small country far from the first step", () => {
        expect(priceNote({slowdown: 1, share: 0.02, next: {share: 0.1, slowdown: 2}})).toBeUndefined()
    })

    it("warns a country close to the first step", () => {
        expect(priceNote({slowdown: 1, share: 0.085, next: {share: 0.1, slowdown: 2}})).toEqual({
            kind: "nearing",
            next: {share: 0.1, slowdown: 2},
        })
    })

    it("says why a big country refills slowly, and what comes next", () => {
        expect(priceNote({slowdown: 1.5, share: 0.6, next: {share: 0.7, slowdown: 2}})).toEqual({
            kind: "slower",
            slowdown: 1.5,
            next: {share: 0.7, slowdown: 2},
        })
    })

    it("names no next step at the top", () => {
        expect(priceNote({slowdown: 2, share: 0.95})).toEqual({kind: "slower", slowdown: 2, next: undefined})
    })
})

describe("factor", () => {
    it("keeps a factor to two decimals, the way the config writes it", () => {
        expect(factor(2)).toBe(2)
        expect(factor(1.5)).toBe(1.5)
        expect(factor(1.25)).toBe(1.25)
        expect(factor(1.3333)).toBe(1.33)
    })
})

describe("shareShown", () => {
    it("rounds down, so a step is never claimed early", () => {
        expect(shareShown(0.0999)).toBeCloseTo(0.099, 10)
        expect(shareShown(0.2499)).toBeCloseTo(0.24, 10)
        expect(shareShown(0.8)).toBeCloseTo(0.8, 10)
    })
})
