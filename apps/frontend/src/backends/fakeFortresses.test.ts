import {describe, expect, it} from "vitest"
import {FakeFortresses} from "./fakeFortresses.ts"

// Landmass 1 is tiles 1 and 2; landmass 2 is tile 3.
const assignment = new Uint16Array([1, 1, 2])

describe("FakeFortresses", () => {
    it("locks a landmass that starts whole to its holder, as the server does at boot", () => {
        const owners: Record<number, string> = {1: "fr", 2: "fr", 3: "de"}
        const fortresses = new FakeFortresses(assignment, 3, (tile) => owners[tile])

        expect(fortresses.fortresses()).toEqual(new Map([[1, "fr"], [2, "de"]]))
        expect(fortresses.fortify(1, "fr")).toBeUndefined()
    })

    it("fortifies a landmass taken whole by another flag, then never twice in a row", () => {
        const owners: Record<number, string> = {1: "fr", 2: "fr", 3: "de"}
        const fortresses = new FakeFortresses(assignment, 3, (tile) => owners[tile])

        fortresses.moved(1, "fr", "es")
        expect(fortresses.fortify(1, "es")).toBeUndefined()
        fortresses.moved(2, "fr", "es")
        const fortified = fortresses.fortify(2, "es")

        expect(fortified?.landmass).toBe(1)
        expect([...fortified!.tiles]).toEqual([1, 2])
        expect(fortresses.fortify(2, "es")).toBeUndefined()
    })
})
