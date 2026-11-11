import {describe, expect, it} from "vitest"
import {capitalsOf} from "./capitals.ts"
import {bridgesOf} from "./bridges.ts"

describe("capitalsOf", () => {
    it("names each capital tile by its country", () => {
        const capitals = capitalsOf({tiles: 30, capitals: {fr: 10, de: 20}}, 30)

        expect(capitals).toEqual(new Map([[10, "fr"], [20, "de"]]))
    })

    it("refuses capitals written for another map", () => {
        expect(() => capitalsOf({tiles: 31, capitals: {fr: 10}}, 30)).toThrow(/30 tiles/)
    })

    it("refuses a capital off the map", () => {
        expect(() => capitalsOf({tiles: 30, capitals: {fr: 31}}, 30)).toThrow(/fr/)
        expect(() => capitalsOf({tiles: 30, capitals: {fr: 0}}, 30)).toThrow(/fr/)
    })
})

describe("bridgesOf", () => {
    it("knows the far end of each bridge on a tile", () => {
        const bridges = bridgesOf({tiles: 30, bridges: [[1, 2], [2, 9]]}, 30)

        expect(bridges.across(2)).toEqual([1, 9])
        expect(bridges.across(9)).toEqual([2])
        expect(bridges.across(5)).toEqual([])
        expect(bridges.all).toEqual([[1, 2], [2, 9]])
    })

    it("refuses bridges written for another map", () => {
        expect(() => bridgesOf({tiles: 31, bridges: []}, 30)).toThrow(/30 tiles/)
    })

    it("refuses a bridge that is not two tiles", () => {
        expect(() => bridgesOf({tiles: 30, bridges: [[1, 31]]}, 30)).toThrow(/bridge 0/)
        expect(() => bridgesOf({tiles: 30, bridges: [[1]]}, 30)).toThrow(/bridge 0/)
        expect(() => bridgesOf({tiles: 30, bridges: [3]}, 30)).toThrow(/bridge 0/)
    })
})
