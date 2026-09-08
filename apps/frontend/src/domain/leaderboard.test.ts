import {beforeEach, describe, expect, it, vi} from "vitest"
import {rankCountries, rankOf} from "./leaderboard.ts"
import {resetWarnOnce} from "./warnOnce.ts"
import {Countries} from "./countries.ts"

const counts = (entries: Record<string, number>) => new Map(Object.entries(entries))
const codes = (entries: Record<string, number>) => rankCountries(counts(entries)).map(e => e.country.code)

beforeEach(() => {
    resetWarnOnce()
    vi.spyOn(console, "warn").mockImplementation(() => {})
})

describe("rankCountries", () => {
    it("ranks by tile count, highest first", () => {
        expect(codes({fr: 3, jp: 10, de: 7})).toEqual(["jp", "de", "fr"])
    })

    it("resolves each code to its country", () => {
        expect(rankCountries(counts({jp: 4}))).toEqual([{country: Countries.get("jp"), tiles: 4}])
    })

    it("returns nothing for an empty map", () => {
        expect(rankCountries(counts({}))).toEqual([])
    })

    /** Otherwise tied countries swap rows on every batch and the table flickers. */
    it("breaks ties on country code so the order is stable", () => {
        expect(codes({jp: 5, fr: 5, de: 5})).toEqual(["de", "fr", "jp"])
        expect(codes({de: 5, jp: 5, fr: 5})).toEqual(["de", "fr", "jp"])
    })

    it("leaves out countries holding no tiles", () => {
        expect(codes({fr: 2, jp: 0, de: -1})).toEqual(["fr"])
    })

    it("leaves out a code with no matching country and says so once", () => {
        expect(codes({fr: 2, zz: 99})).toEqual(["fr"])
        rankCountries(counts({zz: 99}))
        expect(console.warn).toHaveBeenCalledTimes(1)
    })

    it("does not mutate the counts it was given", () => {
        const source = counts({fr: 1, jp: 2})
        rankCountries(source)
        expect(Object.fromEntries(source)).toEqual({fr: 1, jp: 2})
    })
})

describe("rankOf", () => {
    const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})

    it("gives a country's 1-based place in the ranking", () => {
        const ranked = [entry("fr", 500), entry("jp", 250), entry("de", 100)]

        expect(rankOf(ranked, Countries.get("fr")!)).toBe(1)
        expect(rankOf(ranked, Countries.get("de")!)).toBe(3)
    })

    it("gives null for a country that holds no tile", () => {
        expect(rankOf([entry("fr", 500)], Countries.get("jp")!)).toBeNull()
    })

    it("gives null on an empty leaderboard", () => {
        expect(rankOf([], Countries.get("fr")!)).toBeNull()
    })
})
