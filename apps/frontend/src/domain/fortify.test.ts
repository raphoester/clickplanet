import {describe, expect, it} from "vitest"
import {isNews, newsFromTiles} from "./fortify.ts"
import {EARTH_1_SCALE, scaleOf} from "./boardScale.ts"
import {FIXTURE_1} from "../testdata/boards.ts"

const fortification = {landmass: 257, countryId: "bg", tile: 107526, version: {epoch: 1_790_000_000, seq: 9}}

describe("isNews", () => {
    it("is always news to the flag that fortified", () => {
        expect(isNews(fortification, 1, "bg", EARTH_1_SCALE)).toBe(true)
    })

    it("is news to everyone else only from a sizeable territory", () => {
        expect(newsFromTiles(EARTH_1_SCALE)).toBe(50)
        expect(isNews(fortification, 49, "fr", EARTH_1_SCALE)).toBe(false)
        expect(isNews(fortification, 50, "fr", EARTH_1_SCALE)).toBe(true)
    })

    it("counts as much ground on a coarser board", () => {
        expect(newsFromTiles(scaleOf(FIXTURE_1))).toBe(1)
        expect(isNews(fortification, 1, "fr", scaleOf(FIXTURE_1))).toBe(true)
    })
})
