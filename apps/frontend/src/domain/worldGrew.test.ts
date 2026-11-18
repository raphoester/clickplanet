import {describe, expect, it} from "vitest"
import {WORLD_GROWN, worldGrewKept, worldGrewOn} from "./worldGrew.ts"

const earth1 = {name: "earth-1", tiles: 260_503}
const earth2 = {name: "earth-2", tiles: 1_004_682}
const siege1 = {name: "siege-1", tiles: 27_259}

describe("worldGrewOn", () => {
    it("tells a browser that saw the world before, once, how many times the ground it has", () => {
        expect(worldGrewOn(earth2, earth1, null, false)).toEqual({board: "earth-2", times: 4})
        expect(worldGrewOn(earth2, earth1, "earth-1", false)).toEqual({board: "earth-2", times: 4})
        expect(worldGrewOn(earth2, earth1, WORLD_GROWN, false)).toBeUndefined()
    })

    it("tells nothing on a first visit, on the board before, or on an arena's board", () => {
        expect(worldGrewOn(earth2, earth1, null, true)).toBeUndefined()
        expect(worldGrewOn(earth1, earth1, null, false)).toBeUndefined()
        expect(worldGrewOn(siege1, earth1, null, false)).toBeUndefined()
    })
})

describe("worldGrewKept", () => {
    it("keeps the new board as seen on a first visit to it, so the browser is never told later", () => {
        expect(worldGrewKept(earth2, null, true)).toBe(true)
        expect(worldGrewKept(earth2, WORLD_GROWN, true)).toBe(false)
        expect(worldGrewKept(earth2, null, false)).toBe(false)
        expect(worldGrewKept(siege1, null, true)).toBe(false)
    })
})
