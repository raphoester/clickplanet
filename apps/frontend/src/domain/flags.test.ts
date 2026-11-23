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
        const catalog = FlagCatalog.BUILT_IN.withList([{code: "fr", name: "France", closed: false}, BAVARIA])

        expect(catalog.knows(BAVARIA.code)).toBe(true)
        expect(catalog.playable(BAVARIA.code)).toBe(true)
        expect(catalog.openedName(BAVARIA.code)).toBe("Bavaria")
        expect(catalog.playableCodes().at(-1)).toBe(BAVARIA.code)
        expect(catalog.playableCodes()).toHaveLength(Countries.size + 1)
    })

    it("leaves a built-in flag's name to the page's language", () => {
        const catalog = FlagCatalog.BUILT_IN.withList([{code: "fr", name: "Frankreich", closed: false}])
        expect(catalog.openedName("fr")).toBeUndefined()
    })

    it("knows a closed flag, which history names, but lets nobody play it", () => {
        const catalog = FlagCatalog.BUILT_IN.withList([{code: "de", name: "Germany", closed: true}, SAVOY])

        for (const code of ["de", SAVOY.code]) {
            expect(catalog.knows(code)).toBe(true)
            expect(catalog.playable(code)).toBe(false)
            expect(catalog.playableCodes()).not.toContain(code)
        }
        expect(catalog.openedName(SAVOY.code)).toBe("Savoy")
    })

    it("keeps every built-in flag when the list is empty, as a server from before the flags module answers", () => {
        expect(FlagCatalog.BUILT_IN.withList([]).playableCodes()).toEqual([...Countries.keys()])
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
