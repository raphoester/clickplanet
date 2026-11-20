import {describe, expect, it} from "vitest"
import {Attacked, Empire, FarLand, OwnLand} from "../domain/clip/lists.ts"
import {numberOf, percentOf, photosOf} from "./carousel.ts"

const ownLand: OwnLand[] = [
    {country: "fr", tiles: 9000, ownShare: 0, holder: "pw", holderShare: 0.819},
    {country: "gb", tiles: 8000, ownShare: 0.01, holder: "fr", holderShare: 0.6},
    {country: "in", tiles: 6300, ownShare: 0.02, holder: "fr", holderShare: 0.97},
    {country: "de", tiles: 5000, ownShare: 0.05, holder: "br", holderShare: 0.5},
    {country: "br", tiles: 20000, ownShare: 0.1, holder: "ie", holderShare: 0.042},
]

const ranks = [
    {code: "fr", line: "HELD BY PALAU"}, {code: "gb", line: "HELD BY FRANCE"}, {code: "in", line: "HELD BY FRANCE"},
    {code: "de", line: "HELD BY BRAZIL"}, {code: "br", line: "HELD BY IRELAND"},
]

describe("a carousel's photos", () => {
    const photos = photosOf("own-land", ownLand, ranks, "THESE 5 COUNTRIES DON'T OWN THEIR OWN LAND")

    it("are the cover, the ranks from the 5th to the 1st, and the last", () => {
        expect(photos.map((photo) => photo.kind === "rank" ? photo.rank : photo.kind)).toEqual(["cover", 5, 4, 3, 2, 1, "last"])
        expect(photos[0]).toEqual({kind: "cover", hook: "THESE 5 COUNTRIES DON'T OWN THEIR OWN LAND"})
    })

    it("give each rank its flag, its name as people say it, its one number and the editor's words", () => {
        expect(photos[5]).toEqual({kind: "rank", rank: 1, code: "fr", name: "FRANCE", number: "81%", line: "HELD BY PALAU"})
        expect(photos[4]).toMatchObject({rank: 2, name: "THE UK", number: "60%"})
        expect(photos[1]).toMatchObject({rank: 5, number: "4.2%"})
    })

    it("say no address before the last", () => {
        const said = JSON.stringify(photos.slice(0, -1))
        expect(said).not.toContain("clickplanet")
        expect(photos.at(-1)).toEqual({kind: "last"})
    })

    it("are refused when the ranks are not the list's first 5 in its order", () => {
        expect(() => photosOf("own-land", ownLand, [ranks[1], ranks[0], ...ranks.slice(2)], "A COVER")).toThrow(/rank 1 of own-land is "fr", not "gb"/)
        expect(() => photosOf("own-land", ownLand, ranks.slice(0, 4), "A COVER")).toThrow(/5 ranks/)
        expect(() => photosOf("own-land", ownLand.slice(0, 4), ranks, "A COVER")).toThrow(/5 ranks/)
    })
})

describe("a rank's number", () => {
    it("is the holder's share, the tiles taken, the distance, or the share of the world", () => {
        const attacked: Attacked = {flag: "tr", lost: 5300, takenBy: "fr", takenByTiles: 5100, country: "tr"}
        const far: FarLand = {flag: "hu", country: "id", tiles: 2100, km: 10400}
        const empire: Empire = {flag: "pw", tiles: 74000, share: 0.284, country: "ru"}

        expect(numberOf(ownLand[0])).toBe("81%")
        expect(numberOf(attacked)).toBe("5,300 TILES")
        expect(numberOf({...attacked, lost: 1})).toBe("1 TILE")
        expect(numberOf(far)).toBe("10,400 KM")
        expect(numberOf(empire)).toBe("28%")
    })

    it("never says more than it is, to a tenth of a point under a tenth", () => {
        expect(percentOf(0.999)).toBe("99%")
        expect(percentOf(0.0999)).toBe("9.9%")
        expect(percentOf(0)).toBe("0.0%")
    })
})
