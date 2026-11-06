// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {BONUS_GUIDE_STORAGE_KEY} from "../../domain/bonusGuide.ts"

afterEach(() => {
    window.localStorage.clear()
    vi.resetModules()
})

const loaded = async () => (await import("./bonusGuideStore.ts")).useBonusGuide

describe("the bonus guide", () => {
    it("starts from what this browser saved", async () => {
        window.localStorage.setItem(BONUS_GUIDE_STORAGE_KEY, JSON.stringify({won: ["bomb"], uses: {bomb: 2}}))

        expect((await loaded()).getState().guide).toEqual({won: ["bomb"], uses: {bomb: 2}})
    })

    it("saves each use and win in this browser", async () => {
        const guide = await loaded()

        guide.getState().markUsed("refill")
        guide.getState().markWon("refill")

        expect(JSON.parse(window.localStorage.getItem(BONUS_GUIDE_STORAGE_KEY)!)).toEqual({won: ["refill"], uses: {refill: 1}})
    })
})
