import {describe, expect, it} from "vitest"
import {arenaFlagsOf, sameArenaFlags, stands} from "./standing.ts"

describe("the flags still in an arena", () => {
    const arena = arenaFlagsOf(new Map([[1, "fr"], [2, "de"], [3, "it"]]), new Set(["de"]))

    it("are the capitals' countries the arena did not knock out", () => {
        expect(stands(arena, "fr")).toBe(true)
        expect(stands(arena, "de")).toBe(false)
    })

    it("leave out a flag with no capital", () => {
        expect(stands(arena, "gb-eng")).toBe(false)
    })

    it("are every flag outside an arena", () => {
        expect(stands(undefined, "gb-eng")).toBe(true)
    })

    it("are the same when they hold the same flags", () => {
        const again = arenaFlagsOf(new Map([[3, "it"], [2, "de"], [1, "fr"]]), new Set(["de"]))
        expect(sameArenaFlags(arena, again)).toBe(true)
        expect(sameArenaFlags(arena, arenaFlagsOf(new Map([[1, "fr"]]), new Set()))).toBe(false)
        expect(sameArenaFlags(arena, {...arena, out: new Set()})).toBe(false)
        expect(sameArenaFlags(undefined, arena)).toBe(false)
        expect(sameArenaFlags(undefined, undefined)).toBe(true)
    })
})
