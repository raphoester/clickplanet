import {describe, expect, it} from "vitest"
import {outcomeOf, placementOf, ShieldChange, TileShields} from "./shields.ts"

const counts = (shields: TileShields, tiles: number[]) => tiles.map((tile) => shields.shieldsOf(tile))

describe("outcomeOf", () => {
    it("changes nothing on a tile the flag already holds, shielded or not", () => {
        expect(outcomeOf("fr", "fr", 0)).toBe("unchanged")
        expect(outcomeOf("fr", "fr", 4)).toBe("unchanged")
    })

    it("is shielded by the shields standing on another flag's tile, rather than taken", () => {
        expect(outcomeOf("jp", "fr", 1)).toBe("shielded")
    })

    it("takes an unshielded tile, and an empty one", () => {
        expect(outcomeOf("jp", "fr", 0)).toBe("taken")
        expect(outcomeOf(undefined, "fr", 0)).toBe("taken")
    })
})

describe("placementOf", () => {
    it("places a shield on a tile the flag holds", () => {
        expect(placementOf("fr", "fr", 3, 10)).toBe("place")
    })

    it("sends nothing to a tile already at its most", () => {
        expect(placementOf("fr", "fr", 10, 10)).toBe("full")
    })

    it("clicks any tile the flag does not hold", () => {
        expect(placementOf("jp", "fr", 0, 10)).toBe("click")
        expect(placementOf(undefined, "fr", 0, 10)).toBe("click")
    })

    it("leaves the most to the server before the rules say one", () => {
        expect(placementOf("fr", "fr", 40, undefined)).toBe("place")
        expect(placementOf("fr", "fr", 40, 0)).toBe("place")
    })
})

describe("TileShields", () => {
    it("starts every tile with none, and reports how a count moved", () => {
        const shields = new TileShields(10)
        const changes: ShieldChange[] = []

        shields.set(3, 2, changes)
        shields.set(3, 2, changes)
        shields.set(3, -1, changes)

        expect(changes).toEqual([{tile: 3, shields: 2, was: 0}, {tile: 3, shields: 0, was: 2}])
        expect(counts(shields, [1, 3])).toEqual([0, 0])
    })

    it("ignores tiles outside the map", () => {
        const shields = new TileShields(10)
        const changes: ShieldChange[] = []

        shields.set(0, 1, changes)
        shields.set(11, 1, changes)

        expect(changes).toEqual([])
        expect(shields.shieldsOf(11)).toBe(0)
    })
})
