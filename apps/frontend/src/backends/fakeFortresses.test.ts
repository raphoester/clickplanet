import {describe, expect, it} from "vitest"
import {FakeFortresses} from "./fakeFortresses.ts"
import {FakeTally} from "./fakeTally.ts"

// Landmass 1 is tiles 1, 2 and 3: province 1 is tiles 1 and 2, province 2 tile 3. Landmass 2 is tile 4, province 3.
const landmasses = {assignment: new Uint16Array([1, 1, 1, 2]), count: 3}
const provinces = {assignment: new Uint16Array([1, 1, 2, 3]), count: 4}

function fortressesOf(owners: Record<number, string>): {tally: FakeTally, fortresses: FakeFortresses} {
    const tally = new FakeTally(landmasses, provinces)
    for (const [tile, owner] of Object.entries(owners)) tally.moved(Number(tile), undefined, owner)
    return {tally, fortresses: new FakeFortresses(provinces, tally, (tile) => owners[tile])}
}

describe("FakeFortresses", () => {
    it("locks a province that starts whole to its holder, as the server does at boot", () => {
        const {fortresses} = fortressesOf({1: "fr", 2: "fr", 3: "de", 4: "de"})

        expect(fortresses.fortresses()).toEqual(new Map([[1, "fr"], [2, "de"], [3, "de"]]))
        expect(fortresses.fortify(1, "fr")).toBeUndefined()
    })

    it("fortifies a province taken whole by another flag, though its landmass is not, then never twice in a row", () => {
        const {tally, fortresses} = fortressesOf({1: "fr", 2: "fr", 3: "de"})

        tally.moved(1, "fr", "es")
        expect(fortresses.fortify(1, "es")).toBeUndefined()
        tally.moved(2, "fr", "es")
        const fortified = fortresses.fortify(2, "es")

        expect(fortified?.province).toBe(1)
        expect([...fortified!.tiles]).toEqual([1, 2])
        expect(fortresses.fortify(2, "es")).toBeUndefined()
    })
})
