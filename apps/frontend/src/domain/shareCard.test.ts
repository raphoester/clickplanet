import {describe, expect, it} from "vitest"
import {
    cardLayout,
    cardScale,
    cardSize,
    cropToAspect,
    fitInBox,
    shareFileName,
    shareLabel,
    shareStats,
    shareText,
    shareUrl,
    statsLine,
} from "./shareCard.ts"
import {Countries} from "./countries.ts"
import {LeaderboardEntry} from "./leaderboard.ts"

const france = Countries.get("fr")!
const japan = Countries.get("jp")!
const entry = (code: string, tiles: number): LeaderboardEntry => ({country: Countries.get(code)!, tiles})

describe("the share link", () => {
    // A sibling Worker serves the per-country link preview off this exact
    // parameter, so the name and the code spelling are a contract, not a detail.
    it("points at the country the player is holding, by the code the atlas uses", () => {
        expect(shareUrl("fr")).toBe("https://clickplanet.lol/?c=fr")
    })

    it("drops the scheme where it is drawn into the image for a person to read", () => {
        expect(shareLabel("jp")).toBe("clickplanet.lol/?c=jp")
    })

    it("names the file after the country, so a folder of them is readable", () => {
        expect(shareFileName("fr")).toBe("clickplanet-fr.png")
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
})

describe("the text that rides with the image", () => {
    it("carries the standing and the link", () => {
        expect(shareText({country: france, rank: 2, tiles: 9001}))
            .toBe("France is #2 on ClickPlanet with 9,001 tiles. Come and take them. https://clickplanet.lol/?c=fr")
    })

    it("invites rather than reports when there is no rank to quote", () => {
        expect(shareText({country: japan, rank: null, tiles: 0}))
            .toBe("Japan holds nothing on ClickPlanet yet. Come and claim it. https://clickplanet.lol/?c=jp")
    })
})

describe("reading the player's standing off the board", () => {
    it("takes the rank from the row's place and the count from the row", () => {
        expect(shareStats([entry("jp", 500), entry("fr", 250)], france))
            .toEqual({country: france, rank: 2, tiles: 250})
    })

    it("reports no rank for a country that is not on the board at all", () => {
        expect(shareStats([entry("jp", 500)], france))
            .toEqual({country: france, rank: null, tiles: 0})
    })
})

describe("the shape the card comes out in", () => {
    const ratio = ({width, height}: {width: number, height: number}) => width / height

    // A phone's canvas is a 1:2.2 column. Posted, a timeline either shows it as
    // a sliver or crops it itself, which is the thing worth not leaving to it.
    it("takes the middle out of a phone's column rather than posting the column", () => {
        const crop = cropToAspect(390, 844)

        expect(ratio(crop)).toBeCloseTo(9 / 16, 2)
        expect(crop.width).toBe(390)
        expect(crop.height).toBeLessThan(844)
    })

    // The globe is centred in the canvas — the camera looks at the origin — so
    // an off-centre crop would take the planet's head off.
    it("crops evenly, so the globe stays in the middle", () => {
        const crop = cropToAspect(390, 844)

        expect(crop.y).toBe(Math.round((844 - crop.height) / 2))
        expect(crop.x).toBe(0)
    })

    it("catches an ultrawide doing the same thing the other way", () => {
        const crop = cropToAspect(2560, 1080)

        expect(ratio(crop)).toBeCloseTo(16 / 9, 2)
        expect(crop.height).toBe(1080)
        expect(crop.x).toBe(Math.round((2560 - crop.width) / 2))
    })

    it("leaves a shape already worth posting alone", () => {
        expect(cropToAspect(1920, 1080)).toEqual({x: 0, y: 0, width: 1920, height: 1080})
        expect(cropToAspect(1024, 768)).toEqual({x: 0, y: 0, width: 1024, height: 768})
    })

    it("stays on its feet against a canvas with no pixels in it", () => {
        expect(cropToAspect(0, 0)).toEqual({x: 0, y: 0, width: 0, height: 0})
    })

    it("sizes the card from what the crop kept, not from the whole canvas", () => {
        const layout = cardLayout(390, 844)

        expect(ratio(layout)).toBeCloseTo(9 / 16, 2)
        expect(Math.min(layout.width, layout.height)).toBeGreaterThanOrEqual(720)
        expect(layout.crop.height).toBeLessThan(844)
    })
})

describe("the size the card comes out at", () => {
    // The canvas is sized in CSS pixels, so a desktop already captures something
    // worth posting and reprocessing it would only soften it.
    it("leaves a desktop capture at the size it was drawn", () => {
        expect(cardSize(1920, 1080)).toEqual({width: 1920, height: 1080})
    })

    it("lifts a phone capture to something that is not a thumbnail", () => {
        const size = cardSize(390, 693)

        expect(Math.min(size.width, size.height)).toBeGreaterThanOrEqual(720)
        expect(size.width / size.height).toBeCloseTo(390 / 693, 2)
    })

    it("never interpolates further than it is worth", () => {
        expect(cardScale(200, 200)).toBe(2)
    })

    it("brings a very large capture back down to something a share sheet will take", () => {
        const size = cardSize(3840, 2160)

        expect(Math.max(size.width, size.height)).toBeLessThanOrEqual(2400)
        expect(size.width / size.height).toBeCloseTo(3840 / 2160, 2)
    })

    it("stays on its feet against a canvas with no pixels in it", () => {
        expect(cardScale(0, 0)).toBe(1)
    })
})

describe("fitting a flag in its box", () => {
    // The flags are every aspect ratio there is, and a stretched one is the
    // wrong flag — same rule the leaderboard's sprites follow.
    it("fits a wide flag on its width", () => {
        expect(fitInBox({width: 100, height: 50}, {width: 90, height: 60}))
            .toEqual({width: 90, height: 45})
    })

    it("fits a tall flag on its height", () => {
        expect(fitInBox({width: 50, height: 100}, {width: 90, height: 60}))
            .toEqual({width: 30, height: 60})
    })

    it("draws nothing for a region with no area", () => {
        expect(fitInBox({width: 0, height: 0}, {width: 90, height: 60}))
            .toEqual({width: 0, height: 0})
    })
})
