import {afterEach, describe, expect, it, vi} from "vitest"
import {LEAVER_STORAGE_KEY, localLeaverStore} from "./leaverStore.ts"

function stubStorage(kept: Record<string, string> = {}) {
    vi.stubGlobal("window", {
        localStorage: {
            getItem: (key: string) => kept[key] ?? null,
            setItem: (key: string, value: string) => {
                kept[key] = value
            },
            removeItem: (key: string) => {
                delete kept[key]
            },
        },
    })
}

describe("localLeaverStore", () => {
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it("reads back what it wrote, and nothing once it wrote none", () => {
        stubStorage()
        const store = localLeaverStore()

        store.write({user: "cg-42", at: 1000})
        expect(store.read()).toEqual({user: "cg-42", at: 1000})

        store.write(undefined)
        expect(store.read()).toBeUndefined()
    })

    it("reads nothing from something that is not a kept leaver", () => {
        stubStorage({[LEAVER_STORAGE_KEY]: '{"user": 5}'})

        expect(localLeaverStore().read()).toBeUndefined()
    })

    it("reads nothing and writes nothing when the storage is out of reach", () => {
        vi.stubGlobal("window", {})
        const store = localLeaverStore()

        expect(() => store.write({user: "cg-42", at: 1000})).not.toThrow()
        expect(store.read()).toBeUndefined()
    })
})
