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

        store.write({tag: "rc", referrer: "discord.com"})

        expect(store.read()).toEqual({tag: "rc", referrer: "discord.com"})
    })

    it("reads nothing from something that is not a kept landing", () => {
        stubStorage({[LANDING_STORAGE_KEY]: '{"tag": 5}'})

        expect(localLandingStore().read()).toBeUndefined()
    })

    it("reads nothing, and writing does not throw, when storage refuses", () => {
        const refuse = () => {
            throw new Error("the storage is not available")
        }
        vi.stubGlobal("window", {localStorage: {getItem: refuse, setItem: refuse}})
        const store = localLandingStore()

        expect(() => store.write({tag: "rc", referrer: "discord.com"})).not.toThrow()
        expect(store.read()).toBeUndefined()
    })
})
