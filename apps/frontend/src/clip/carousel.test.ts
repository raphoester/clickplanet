import {describe, expect, it} from "vitest"
import {Empire, FarLand, Fighting} from "../domain/clip/lists.ts"
import {numberOf, percentOf, photosOf} from "./carousel.ts"

const fighting: Fighting[] = [
    {flag: "fr", fought: 41200, took: 30000, lost: 11200, rival: "pw", rivalTiles: 9000, country: "fr"},
    {flag: "gb", fought: 20000, took: 9000, lost: 11000, rival: "fr", rivalTiles: 8000, country: "gb"},
    {flag: "in", fought: 9000, took: 4000, lost: 5000, rival: "fr", rivalTiles: 3000, country: "in"},
    {flag: "de", fought: 5000, took: 2500, lost: 2500, rival: "br", rivalTiles: 2000, country: "de"},
    {flag: "br", fought: 1, took: 1, lost: 0, rival: "ie", rivalTiles: 1, country: "br"},
]

const ranks = [
    {code: "fr", line: "MOSTLY AGAINST PALAU"}, {code: "gb", line: "MOSTLY AGAINST FRANCE"}, {code: "in", line: "MOSTLY AGAINST FRANCE"},
    {code: "de", line: "MOSTLY AGAINST BRAZIL"}, {code: "br", line: "MOSTLY AGAINST IRELAND"},
]

describe("a carousel's photos", () => {
    const photos = photosOf("fight-the-most", fighting, ranks, "THE 5 COUNTRIES THAT FIGHT THE MOST")

    it("are the cover, the ranks from the 5th to the 1st, and the last", () => {
        expect(photos.map((photo) => photo.kind === "rank" ? photo.rank : photo.kind)).toEqual(["cover", 5, 4, 3, 2, 1, "last"])
        expect(photos[0]).toEqual({kind: "cover", hook: "THE 5 COUNTRIES THAT FIGHT THE MOST"})
    })

    it("give each rank its flag, its name as people say it, its one number and the editor's words", () => {
        expect(photos[5]).toEqual({kind: "rank", rank: 1, code: "fr", name: "FRANCE", number: "41,200 TILES FOUGHT", line: "MOSTLY AGAINST PALAU"})
        expect(photos[4]).toMatchObject({rank: 2, name: "THE UK", number: "20,000 TILES FOUGHT"})
        expect(photos[1]).toMatchObject({rank: 5, number: "1 TILE FOUGHT"})
    })

    it("say no address before the last", () => {
        const said = JSON.stringify(photos.slice(0, -1))
        expect(said).not.toContain("clickplanet")
        expect(photos.at(-1)).toEqual({kind: "last"})
    })

    it("are refused when the ranks are not the list's first 5 in its order", () => {
        expect(() => photosOf("fight-the-most", fighting, [ranks[1], ranks[0], ...ranks.slice(2)], "A COVER")).toThrow(/rank 1 of fight-the-most is "fr", not "gb"/)
        expect(() => photosOf("fight-the-most", fighting, ranks.slice(0, 4), "A COVER")).toThrow(/5 ranks/)
        expect(() => photosOf("fight-the-most", fighting.slice(0, 4), ranks, "A COVER")).toThrow(/5 ranks/)
    })
})

describe("a rank's number", () => {
    it("is the tiles fought, the distance, or the share of the world", () => {
        const far: FarLand = {flag: "hu", country: "id", tiles: 2100, km: 10400}
        const empire: Empire = {flag: "pw", tiles: 74000, share: 0.284, country: "ru"}

        expect(numberOf(fighting[0])).toBe("41,200 TILES FOUGHT")
        expect(numberOf(far)).toBe("10,400 KM")
        expect(numberOf(empire)).toBe("28%")
    })

    it("never says more than it is, to a tenth of a point under a tenth", () => {
        expect(percentOf(0.999)).toBe("99%")
        expect(percentOf(0.0999)).toBe("9.9%")
        expect(percentOf(0)).toBe("0.0%")
    })
})
