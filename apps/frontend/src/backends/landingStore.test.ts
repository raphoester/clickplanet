import {afterEach, describe, expect, it, vi} from "vitest"
import {LANDING_STORAGE_KEY, localLandingStore} from "./landingStore.ts"

function stubStorage(kept: Record<string, string> = {}) {
    vi.stubGlobal("window", {
        localStorage: {
            getItem: (key: string) => kept[key] ?? null,
            setItem: (key: string, value: string) => {
                kept[key] = value
            },
        },
    })
}

describe("localLandingStore", () => {
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it("reads back what it wrote", () => {
        stubStorage()
        const store = localLandingStore()

        store.write({flag: "fr", tag: ""})

        expect(store.read()).toEqual({flag: "fr", tag: ""})
    })

    it("reads nothing from something that is not a kept landing", () => {
        stubStorage({[LANDING_STORAGE_KEY]: '{"flag": 5}'})

        expect(localLandingStore().read()).toBeUndefined()
    })

    it("reads nothing, and writing does not throw, when storage refuses", () => {
        const refuse = () => {
            throw new Error("the storage is not available")
        }
        vi.stubGlobal("window", {localStorage: {getItem: refuse, setItem: refuse}})
        const store = localLandingStore()

        expect(() => store.write({flag: "fr", tag: ""})).not.toThrow()
        expect(store.read()).toBeUndefined()
    })
})
