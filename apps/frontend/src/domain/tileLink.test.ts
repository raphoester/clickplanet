import {describe, expect, it} from "vitest"
import {linkedTile, withoutLinkedTile} from "./tileLink.ts"

describe("linkedTile", () => {
    it("reads the tile a link opens on", () => {
        expect(linkedTile(new URL("https://clickplanet.lol/?tile=4242"))).toBe(4242)
        expect(linkedTile(new URL("https://clickplanet.lol/play?f=fr&tile=%2042%20"))).toBe(42)
    })

    it("reads nothing that is not a tile", () => {
        for (const search of ["", "?tile=", "?tile=0", "?tile=-3", "?tile=1.5", "?tile=abc", "?tile=99999999999"]) {
            expect(linkedTile(new URL(`https://clickplanet.lol/${search}`)), search).toBeUndefined()
        }
    })
})

describe("withoutLinkedTile", () => {
    it("drops the tile and keeps the rest", () => {
        expect(withoutLinkedTile(new URL("https://clickplanet.lol/play?f=fr&tile=42")).href)
            .toBe("https://clickplanet.lol/play?f=fr")
    })
})
