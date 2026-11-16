import {describe, expect, it} from "vitest"
import {InfectionChange, infectionMarksOf, TileInfections} from "./infections.ts"

describe("infectionMarksOf", () => {
    it("marks each tile as its last change says: a tile taken then cured by the end is unmarked", () => {
        expect(infectionMarksOf([
            {tile: 1, virus: "v-1"}, {tile: 2, virus: "v-1"}, {tile: 1, virus: undefined}, {tile: 3, virus: undefined},
        ])).toEqual({on: [2], off: [1, 3]})
    })
})

describe("TileInfections", () => {
    it("tells each change once, and none for a tile off the map", () => {
        const infections = new TileInfections(4)
        const changes: InfectionChange[] = []

        infections.set(1, "v-1", changes)
        infections.set(1, "v-1", changes)
        infections.set(5, "v-1", changes)
        infections.set(1, undefined, changes)

        expect(changes).toEqual([{tile: 1, virus: "v-1"}, {tile: 1, virus: undefined}])
        expect(infections.count()).toBe(0)
    })

    it("knows every tile of one virus", () => {
        const infections = new TileInfections(4)
        const changes: InfectionChange[] = []
        infections.set(1, "v-1", changes)
        infections.set(2, "v-2", changes)
        infections.set(3, "v-1", changes)

        expect(infections.tilesOf("v-1")).toEqual([1, 3])
        expect(infections.virusOf(2)).toBe("v-2")
    })
})
