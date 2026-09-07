import {describe, expect, it} from "vitest"
import {visibleOptions} from "./visibleOptions.ts"

const VALUES = [
    {code: "fr", name: "France"},
    {code: "jp", name: "Japan"},
    {code: "de", name: "Germany"},
]

describe("visibleOptions", () => {
    const [FRANCE, JAPAN, GERMANY] = VALUES

    it("returns everything when nothing is typed", () => {
        expect(visibleOptions(VALUES, FRANCE, "")).toEqual(VALUES)
        expect(visibleOptions(VALUES, FRANCE, "   ")).toEqual(VALUES)
    })

    it("filters case-insensitively on a substring", () => {
        expect(visibleOptions(VALUES, JAPAN, "AN")).toEqual([FRANCE, JAPAN, GERMANY])
        expect(visibleOptions(VALUES, JAPAN, "rman")).toEqual([JAPAN, GERMANY])
    })

    it("leaves a matching selection where the search put it", () => {
        expect(visibleOptions(VALUES, JAPAN, "jap")).toEqual([JAPAN])
    })

    /** Without this the browser reassigns the selection to the first result. */
    it("keeps a filtered-out selection in the list, ahead of the results", () => {
        expect(visibleOptions(VALUES, FRANCE, "jap")).toEqual([FRANCE, JAPAN])
    })

    it("still offers the selection when the search matches nothing", () => {
        expect(visibleOptions(VALUES, FRANCE, "zzzz")).toEqual([FRANCE])
    })

    /**
     * The invariant the whole component depends on: the controlled value is
     * always one of the options rendered for it.
     */
    it("always contains the selection, for any search over any country", () => {
        for (const selected of VALUES) {
            for (const search of ["", " ", "a", "an", "jap", "france", "zzzz", "GERMANY", "é"]) {
                const options = visibleOptions(VALUES, selected, search)
                expect(options.some(v => v.code === selected.code), `${selected.code} / "${search}"`).toBe(true)
            }
        }
    })

    it("never lists the same country twice", () => {
        for (const search of ["", "a", "jap", "zzzz"]) {
            const codes = visibleOptions(VALUES, JAPAN, search).map(v => v.code)
            expect(new Set(codes).size, search).toBe(codes.length)
        }
    })
})
