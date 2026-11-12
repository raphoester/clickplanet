import {afterEach, describe, expect, it, vi} from "vitest"
import {HANDED_REFERRER_KEY, handedReferrer, LANDING_STORAGE_KEY, localLandingStore} from "./landingStore.ts"

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

    it("reads back the recruiter kept with the landing", () => {
        stubStorage()
        const store = localLandingStore()

        store.write({tag: "rc", referrer: "", recruiterAccountId: "0193f2a4-7c1e-7000-8000-00000000ada0"})

        expect(store.read()).toEqual({tag: "rc", referrer: "", recruiterAccountId: "0193f2a4-7c1e-7000-8000-00000000ada0"})
    })

    it("reads a landing kept before recruiters were, with none", () => {
        stubStorage({[LANDING_STORAGE_KEY]: '{"tag": "rc", "referrer": "discord.com"}'})

        expect(localLandingStore().read()).toEqual({tag: "rc", referrer: "discord.com"})
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

describe("handedReferrer", () => {
    function stubSession(kept: Record<string, string>) {
        vi.stubGlobal("window", {
            sessionStorage: {
                getItem: (key: string) => kept[key] ?? null,
                removeItem: (key: string) => {
                    delete kept[key]
                },
            },
        })
    }

    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it("reads the page before a flag's address once, so a reload of the game is not that page", () => {
        stubSession({[HANDED_REFERRER_KEY]: "https://www.tiktok.com/"})

        expect(handedReferrer()).toBe("https://www.tiktok.com/")
        expect(handedReferrer()).toBeUndefined()
    })

    it("reads an empty one as it was handed: the address was typed", () => {
        stubSession({[HANDED_REFERRER_KEY]: ""})

        expect(handedReferrer()).toBe("")
    })

    it("reads nothing when storage refuses", () => {
        vi.stubGlobal("window", {})

        expect(handedReferrer()).toBeUndefined()
    })
})
