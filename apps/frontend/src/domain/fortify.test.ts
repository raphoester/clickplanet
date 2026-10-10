import {describe, expect, it} from "vitest"
import {isNews, NEWS_FROM_TILES} from "./fortify.ts"

const fortification = {landmass: 257, countryId: "bg", tile: 107526}

describe("isNews", () => {
    it("is always news to the flag that fortified", () => {
        expect(isNews(fortification, 1, "bg")).toBe(true)
    })

    it("is news to everyone else only from a sizeable territory", () => {
        expect(isNews(fortification, NEWS_FROM_TILES - 1, "fr")).toBe(false)
        expect(isNews(fortification, NEWS_FROM_TILES, "fr")).toBe(true)
    })
})
