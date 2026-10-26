import {describe, expect, it} from "vitest"
import {freeTilesOf, rankHoldings, sameTerritory, sharePercent, Territory} from "./territory.ts"

const SICILY: Territory = {
    landmass: 12,
    name: "Sicily",
    ground: "it",
    size: 100,
    holdings: [{flag: "fr", tiles: 60}, {flag: "it", tiles: 30}],
    fortifiedBy: "it",
    target: undefined,
}

describe("rankHoldings", () => {
    it("ranks the flags by the tiles they hold, ties on the flag's code, and leaves out a flag with none", () => {
        const held = new Map([["it", 30], ["de", 60], ["fr", 60], ["es", 0]])

        expect(rankHoldings(held)).toEqual([
            {flag: "de", tiles: 60},
            {flag: "fr", tiles: 60},
            {flag: "it", tiles: 30},
        ])
    })
})

describe("freeTilesOf", () => {
    it("is what no flag holds", () => {
        expect(freeTilesOf(SICILY)).toBe(10)
        expect(freeTilesOf({...SICILY, holdings: []})).toBe(100)
    })
})

describe("sharePercent", () => {
    it("is a whole percent of the territory", () => {
        expect(sharePercent(60, 100)).toBe("60")
        expect(sharePercent(1, 3)).toBe("33")
        expect(sharePercent(100, 100)).toBe("100")
        expect(sharePercent(0, 100)).toBe("0")
    })

    it("never rounds a few tiles down to nothing, nor all but one up to all", () => {
        expect(sharePercent(1, 1_000)).toBe("<1")
        expect(sharePercent(999, 1_000)).toBe("99")
    })
})

describe("sameTerritory", () => {
    it("is the same while nothing on the card would change", () => {
        expect(sameTerritory(SICILY, {...SICILY, holdings: SICILY.holdings.map((holding) => ({...holding}))})).toBe(true)
    })

    it("differs on a tile changing hands, a new lock or a flag getting close", () => {
        expect(sameTerritory(SICILY, {...SICILY, holdings: [{flag: "fr", tiles: 61}, {flag: "it", tiles: 29}]})).toBe(false)
        expect(sameTerritory(SICILY, {...SICILY, holdings: [{flag: "fr", tiles: 60}]})).toBe(false)
        expect(sameTerritory(SICILY, {...SICILY, fortifiedBy: "fr"})).toBe(false)
        expect(sameTerritory(SICILY, {...SICILY, target: {flag: "fr", missing: 25}})).toBe(false)
        expect(sameTerritory({...SICILY, target: {flag: "fr", missing: 25}}, {...SICILY, target: {flag: "fr", missing: 24}})).toBe(false)
    })
})
