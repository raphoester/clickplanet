import {describe, expect, it} from "vitest"
import {FlagCatalog, FlagReads, isOpenedCode, ListedFlag} from "./flags.ts"
import {Countries} from "./countries.ts"

const BAVARIA: ListedFlag = {code: "r-k3p9x2ma0b", name: "Bavaria", closed: false}
const SAVOY: ListedFlag = {code: "r-0a1b2c3d4e", name: "Savoy", closed: true}

describe("isOpenedCode", () => {
    it("is r- and ten letters or digits", () => {
        expect(isOpenedCode("r-k3p9x2ma0b")).toBe(true)
        expect(isOpenedCode("fr")).toBe(false)
        expect(isOpenedCode("gb-eng")).toBe(false)
        expect(isOpenedCode("r-K3P9X2MA0B")).toBe(false)
        expect(isOpenedCode("r-k3p9x2ma0")).toBe(false)
        expect(isOpenedCode("r-k3p9x2ma0bb")).toBe(false)
    })
})

describe("FlagCatalog", () => {
    it("knows the flags the game ships with, every one of them playable, in their order", () => {
        const catalog = FlagCatalog.BUILT_IN
        expect(catalog.knows("fr")).toBe(true)
        expect(catalog.playableCodes()).toEqual([...Countries.keys()])
        expect(catalog.knows(BAVARIA.code)).toBe(false)
    })

    it("learns an opened flag and its English name from the list, after the built-in ones", () => {
        const catalog = FlagCatalog.BUILT_IN.withFlags([{code: "fr", name: "France", closed: false}, BAVARIA])

        expect(catalog.knows(BAVARIA.code)).toBe(true)
        expect(catalog.playable(BAVARIA.code)).toBe(true)
        expect(catalog.openedName(BAVARIA.code)).toBe("Bavaria")
        expect(catalog.playableCodes().at(-1)).toBe(BAVARIA.code)
        expect(catalog.playableCodes()).toHaveLength(Countries.size + 1)
    })

    it("leaves a built-in flag's name to the page's language", () => {
        const catalog = FlagCatalog.BUILT_IN.withFlags([{code: "fr", name: "Frankreich", closed: false}])
        expect(catalog.openedName("fr")).toBeUndefined()
    })

    it("knows a closed flag, which history names, but lets nobody play it", () => {
        const catalog = FlagCatalog.BUILT_IN.withFlags([{code: "de", name: "Germany", closed: true}, SAVOY])

        for (const code of ["de", SAVOY.code]) {
            expect(catalog.knows(code)).toBe(true)
            expect(catalog.playable(code)).toBe(false)
            expect(catalog.playableCodes()).not.toContain(code)
        }
        expect(catalog.openedName(SAVOY.code)).toBe("Savoy")
    })

    it("keeps every built-in flag when the list is empty, as a server from before the flags module answers", () => {
        expect(FlagCatalog.BUILT_IN.withFlags([]).playableCodes()).toEqual([...Countries.keys()])
    })

    it("takes a flag the stream tells over the catalog, in the order told, with no other flag lost", () => {
        const catalog = FlagCatalog.BUILT_IN.withFlags([BAVARIA])
            .withFlags([SAVOY])
            .withFlags([{...BAVARIA, closed: true}])

        expect(catalog.knows(SAVOY.code)).toBe(true)
        expect(catalog.playable(BAVARIA.code)).toBe(false)
        expect(catalog.knows(BAVARIA.code)).toBe(true)

        const reopened = catalog.withFlags([{...BAVARIA, name: "Bayern"}, {...SAVOY, closed: false}])
        expect(reopened.playableCodes().slice(-2)).toEqual([BAVARIA.code, SAVOY.code])
        expect(reopened.openedName(BAVARIA.code)).toBe("Bayern")
    })
})

describe("a sold flag in the catalog", () => {
    const PICTURE = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}

    it("keeps its picture and its time, open or closed, and lists every picture for the globe", () => {
        const sold = {...BAVARIA, sold: {picture: PICTURE, endsAt: 10}}
        const catalog = FlagCatalog.BUILT_IN.withFlags([sold])
        expect(catalog.pictureOf(BAVARIA.code)).toEqual(PICTURE)
        expect(catalog.soldOf(BAVARIA.code)?.endsAt).toBe(10)

        const closed = catalog.withFlags([{...sold, closed: true, sold: {picture: PICTURE, endsAt: 10, closedAt: 11}}])
        expect(closed.closedFlag(BAVARIA.code)).toBe(true)
        expect(closed.pictures()).toEqual(new Map([[BAVARIA.code, PICTURE]]))
        expect(closed.soldOf(BAVARIA.code)?.closedAt).toBe(11)
    })

    it("keeps no picture for a flag of the build, which the atlas draws", () => {
        const catalog = FlagCatalog.BUILT_IN.withFlags([{code: "fr", name: "France", closed: false, sold: {picture: PICTURE, endsAt: 1}}])
        expect(catalog.pictureOf("fr")).toBeUndefined()
        expect(catalog.pictures().size).toBe(0)
    })
})

describe("FlagReads", () => {
    it("reads at load, and once more behind the read in flight when the list changes meanwhile", () => {
        const reads = new FlagReads()
        expect(reads.changed()).toBe(true)
        expect(reads.changed()).toBe(false)
        expect(reads.changed()).toBe(false)

        expect(reads.ended()).toBe(true)
        expect(reads.ended()).toBe(false)
    })

    it("reads once for a code it does not know, and never again for that code", () => {
        const reads = new FlagReads()
        expect(reads.missed("r-k3p9x2ma0b")).toBe(true)
        expect(reads.ended()).toBe(false)

        for (let i = 0; i < 100; i++) expect(reads.missed("r-k3p9x2ma0b")).toBe(false)
    })

    it("lets the read in flight answer for a code missed meanwhile", () => {
        const reads = new FlagReads()
        expect(reads.changed()).toBe(true)
        expect(reads.missed("r-k3p9x2ma0b")).toBe(false)
        expect(reads.ended()).toBe(false)
        expect(reads.missed("r-k3p9x2ma0b")).toBe(false)
    })

    it("reads for another code it does not know", () => {
        const reads = new FlagReads()
        expect(reads.missed("r-k3p9x2ma0b")).toBe(true)
        reads.ended()
        expect(reads.missed("r-0a1b2c3d4e")).toBe(true)
    })

    it("still reads when the list changes, whatever was missed", () => {
        const reads = new FlagReads()
        reads.missed("r-k3p9x2ma0b")
        reads.ended()
        expect(reads.changed()).toBe(true)
    })
})
