import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {PromoBackend, TeaserCard} from "../../backends/promo.ts"
import {useTeaserCards} from "./teaserCardsStore.ts"

const RUSH: TeaserCard[] = [{backgroundUrl: "https://media.test/rush", headline: "Clicks refill 3× faster", line: ""}]

type Call = {key: string, resolve: (cards: TeaserCard[]) => void, reject: (e: unknown) => void}

// A backend whose every call waits for the test to answer it.
function pending(): PromoBackend & {calls: Call[]} {
    const calls: Call[] = []
    return {
        calls,
        teaser: (key) => new Promise((resolve, reject) => calls.push({key, resolve, reject})),
    }
}

const store = () => useTeaserCards.getState()

beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {})
})

afterEach(() => {
    store().serve(undefined)
    vi.restoreAllMocks()
})

describe("teaserCardsStore", () => {
    it("reads a battle's cards once, however often it is asked", async () => {
        const promo = pending()
        store().serve(promo)

        store().ask("tonight")
        store().ask("tonight")
        expect(promo.calls.map((call) => call.key)).toEqual(["tonight"])
        expect(store().cards.tonight).toBeUndefined()

        promo.calls[0].resolve(RUSH)
        await Promise.resolve()
        expect(store().cards.tonight).toEqual(RUSH)

        store().ask("tonight")
        expect(promo.calls).toHaveLength(1)
    })

    it("reads what was asked before promo served it", async () => {
        store().ask("tonight")
        const promo = pending()
        store().serve(promo)

        expect(promo.calls.map((call) => call.key)).toEqual(["tonight"])
    })

    it("keeps none for a read that failed, so nothing is teased", async () => {
        const promo = pending()
        store().serve(promo)

        store().ask("tonight")
        promo.calls[0].reject(new Error("down"))
        await Promise.resolve()

        expect(store().cards.tonight).toEqual([])
        expect(console.error).toHaveBeenCalled()
    })

    it("forgets every card once it stops, and drops a read answered after", async () => {
        const promo = pending()
        store().serve(promo)
        store().ask("tonight")
        store().ask("friday")
        promo.calls[0].resolve(RUSH)
        await Promise.resolve()

        store().serve(undefined)
        expect(store().cards).toEqual({})
        promo.calls[1].resolve(RUSH)
        await Promise.resolve()
        expect(store().cards).toEqual({})
    })

    it("reads nothing for a battle with no key", () => {
        const promo = pending()
        store().serve(promo)

        store().ask("")
        expect(promo.calls).toEqual([])
    })
})
