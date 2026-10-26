import {describe, expect, it} from "vitest"
import {FakeFortresses} from "./fakeFortresses.ts"
import {FakeTally} from "./fakeTally.ts"

// Landmass 1 is tiles 1 and 2; landmass 2 is tile 3.
const assignment = new Uint16Array([1, 1, 2])

function fortressesOf(owners: Record<number, string>): {tally: FakeTally, fortresses: FakeFortresses} {
    const tally = new FakeTally(assignment, 3)
    for (const [tile, owner] of Object.entries(owners)) tally.moved(Number(tile), undefined, owner)
    return {tally, fortresses: new FakeFortresses(assignment, 3, tally, (tile) => owners[tile])}
}

describe("FakeFortresses", () => {
    it("locks a landmass that starts whole to its holder, as the server does at boot", () => {
        const {fortresses} = fortressesOf({1: "fr", 2: "fr", 3: "de"})

        expect(fortresses.fortresses()).toEqual(new Map([[1, "fr"], [2, "de"]]))
        expect(fortresses.fortify(1, "fr")).toBeUndefined()
    })

    it("fortifies a landmass taken whole by another flag, then never twice in a row", () => {
        const {tally, fortresses} = fortressesOf({1: "fr", 2: "fr", 3: "de"})

        tally.moved(1, "fr", "es")
        expect(fortresses.fortify(1, "es")).toBeUndefined()
        tally.moved(2, "fr", "es")
        const fortified = fortresses.fortify(2, "es")

        expect(fortified?.landmass).toBe(1)
        expect([...fortified!.tiles]).toEqual([1, 2])
        expect(fortresses.fortify(2, "es")).toBeUndefined()
    })
})
