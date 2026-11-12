import {describe, expect, it} from "vitest"
import {Countries} from "../../domain/countries.ts"
import {breakHeadline, pictureAlt, shareText, statsLine} from "./shareLines.ts"

const france = Countries.get("fr")!
const japan = Countries.get("jp")!

describe("the invite sent while the map is frozen", () => {
    it("names the next season and the country", () => {
        expect(breakHeadline(france, 1)).toBe("Season 1 starts on a blank map. Play for France on ClickPlanet!")
    })

    it("names the country alone when no season is planned", () => {
        expect(breakHeadline(japan)).toBe("Play for Japan on ClickPlanet!")
    })
})

describe("what the card says", () => {
    it("gives the rank and the count the leaderboard is showing", () => {
        expect(statsLine({country: france, rank: 3, tiles: 12345})).toBe("RANK #3 · 12,345 TILES")
    })

    it("says a single tile as one, not as one tiles", () => {
        expect(statsLine({country: france, rank: 40, tiles: 1})).toBe("RANK #40 · 1 TILE")
    })

    it("says so plainly when the country holds nothing, rather than showing a rank it has not got", () => {
        expect(statsLine({country: france, rank: null, tiles: 0})).toBe("NO TILES YET")
    })

    it("keeps the rank of a country that the season ranks with no ground", () => {
        expect(statsLine({country: france, rank: 2, tiles: 0})).toBe("RANK #2 · NO TILES")
    })

    it("describes the picture with the same standing", () => {
        expect(pictureAlt({country: france, rank: 3, tiles: 12345}))
            .toBe("France on the globe as you have it framed — rank #3 · 12,345 tiles")
        expect(pictureAlt({country: japan, rank: null, tiles: 0}))
            .toBe("Japan on the globe as you have it framed — no tiles yet")
    })
})

describe("the text that rides with the image", () => {
    it("carries the standing and the link", () => {
        expect(shareText({country: france, rank: 2, tiles: 9001}, {code: "fr", name: "Ada Lovelace"}))
            .toBe("France is #2 on ClickPlanet with 9,001 tiles. Come and take them. https://clickplanet.lol/fr/Ada-Lovelace")
    })

    it("invites rather than reports when there is no rank to quote", () => {
        expect(shareText({country: japan, rank: null, tiles: 0}, {code: "jp"}))
            .toBe("Japan holds nothing on ClickPlanet yet. Come and claim it. https://clickplanet.lol/jp")
    })

    it("offers no tiles to take from a country ranked with none", () => {
        expect(shareText({country: france, rank: 2, tiles: 0}, {code: "fr"}))
            .toBe("France is #2 on ClickPlanet with no tiles. Come and claim it. https://clickplanet.lol/fr")
    })
})
