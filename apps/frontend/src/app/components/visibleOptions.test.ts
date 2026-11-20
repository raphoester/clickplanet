import {describe, expect, it} from "vitest"
import {foldForSearch, visibleOptions} from "./visibleOptions.ts"

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

    it("ignores accents on either side", () => {
        const NEPAL = {code: "np", name: "Népal"}
        const IVORY = {code: "ci", name: "Côte d’Ivoire"}
        const names = [FRANCE, NEPAL, IVORY]
        expect(visibleOptions(names, FRANCE, "Nep")).toEqual([FRANCE, NEPAL])
        expect(visibleOptions(names, FRANCE, "nép")).toEqual([FRANCE, NEPAL])
        expect(visibleOptions(names, FRANCE, "NÉPAL")).toEqual([FRANCE, NEPAL])
        expect(visibleOptions(names, FRANCE, "cote d'iv")).toEqual([FRANCE, IVORY])
    })

    it("folds the letters that carry their own mark", () => {
        expect(foldForSearch("Włochy", "pl")).toBe("wlochy")
        expect(foldForSearch("Færøerne", "da")).toBe("faeroerne")
        expect(foldForSearch("Türkiye", "tr")).toBe("turkiye")
        expect(foldForSearch("Irak", "tr")).toBe("irak")
        expect(foldForSearch("İran", "tr")).toBe("iran")
        expect(foldForSearch("İran", "en")).toBe("iran")
        expect(foldForSearch("Çekya", "tr")).toBe("cekya")
    })

    it("leaves a matching selection where the search put it", () => {
        expect(visibleOptions(VALUES, JAPAN, "jap")).toEqual([JAPAN])
    })

    it("keeps a filtered-out selection in the list, ahead of the results", () => {
        expect(visibleOptions(VALUES, FRANCE, "jap")).toEqual([FRANCE, JAPAN])
    })

    it("still offers the selection when the search matches nothing", () => {
        expect(visibleOptions(VALUES, FRANCE, "zzzz")).toEqual([FRANCE])
    })

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

    it("keeps no value out of the list when none is selected", () => {
        expect(visibleOptions(VALUES, undefined, "an")).toEqual([FRANCE, JAPAN, GERMANY])
        expect(visibleOptions(VALUES, undefined, "jap")).toEqual([JAPAN])
    })
})
