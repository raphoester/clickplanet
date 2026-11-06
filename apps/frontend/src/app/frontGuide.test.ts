import {describe, expect, it} from "vitest"
import {COUNTRY_STORAGE_KEY} from "./countryPreference.ts"
import {countGuidedTake, FRONT_GUIDE_STORAGE_KEY, enterFrontGuide, GUIDED_TAKES} from "./frontGuide.ts"

function storeOf(entries: Record<string, string> = {}) {
    const values = new Map(Object.entries(entries))
    return {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => void values.set(key, value),
        values,
    }
}

describe("enterFrontGuide", () => {
    it("guides a first visit, and remembers it", () => {
        const store = storeOf()

        expect(enterFrontGuide(() => store)).toBe(true)
        expect(store.values.get(FRONT_GUIDE_STORAGE_KEY)).toBe("0")
    })

    it("leaves alone a browser that played before the guide", () => {
        const store = storeOf({[COUNTRY_STORAGE_KEY]: JSON.stringify({code: "fr", name: "France"})})

        expect(enterFrontGuide(() => store)).toBe(false)
        expect(store.values.has(FRONT_GUIDE_STORAGE_KEY)).toBe(false)
    })

    it("keeps guiding the next pages until the guided takes are made", () => {
        const country = {[COUNTRY_STORAGE_KEY]: JSON.stringify({code: "fr", name: "France"})}

        const guiding = storeOf({...country, [FRONT_GUIDE_STORAGE_KEY]: String(GUIDED_TAKES - 1)})
        const done = storeOf({...country, [FRONT_GUIDE_STORAGE_KEY]: String(GUIDED_TAKES)})

        expect(enterFrontGuide(() => guiding)).toBe(true)
        expect(enterFrontGuide(() => done)).toBe(false)
    })

    it("guides nothing when the storage refuses", () => {
        expect(enterFrontGuide(() => {
            throw new Error("denied")
        })).toBe(false)
    })
})

describe("countGuidedTake", () => {
    it("counts up to the guided takes and no further", () => {
        const store = storeOf({[FRONT_GUIDE_STORAGE_KEY]: String(GUIDED_TAKES - 1)})

        countGuidedTake(() => store)
        countGuidedTake(() => store)

        expect(store.values.get(FRONT_GUIDE_STORAGE_KEY)).toBe(String(GUIDED_TAKES))
    })

    it("counts nothing for a browser that was never guided", () => {
        const store = storeOf()

        countGuidedTake(() => store)

        expect(store.values.has(FRONT_GUIDE_STORAGE_KEY)).toBe(false)
    })
})
