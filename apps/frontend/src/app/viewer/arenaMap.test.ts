import {describe, expect, it} from "vitest"
import {capitalsOf} from "./capitals.ts"
import {watersOf} from "./waters.ts"

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

describe("watersOf", () => {
    it("knows each body's coast and the bodies each coast tile is on", () => {
        const waters = watersOf({tiles: 30, bodies: [[1, 2, 3], [3, 4]]}, 30)

        expect(waters.coastOf(0)).toEqual([1, 2, 3])
        expect(waters.bodiesOf(3)).toEqual([0, 1])
        expect(waters.bodiesOf(5)).toEqual([])
    })

    it("refuses waters written for another map", () => {
        expect(() => watersOf({tiles: 31, bodies: []}, 30)).toThrow(/30 tiles/)
    })

    it("refuses a coast that is not a list of tiles", () => {
        expect(() => watersOf({tiles: 30, bodies: [[1, 31]]}, 30)).toThrow(/water 0/)
        expect(() => watersOf({tiles: 30, bodies: [3]}, 30)).toThrow(/water 0/)
    })
})
