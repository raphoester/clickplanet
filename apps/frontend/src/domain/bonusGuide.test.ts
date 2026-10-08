import {describe, expect, it} from "vitest"
import {
    afterUse,
    afterWin,
    FRESH_GUIDE,
    isFirstWin,
    isLearning,
    isNew,
    LEARNING_USES,
    parseBonusGuide,
    usesOf,
} from "./bonusGuide.ts"

describe("parseBonusGuide", () => {
    it("starts fresh with nothing stored", () => {
        expect(parseBonusGuide(null)).toEqual(FRESH_GUIDE)
    })

    it("starts fresh from what is not a guide", () => {
        for (const raw of ["", "not json", "null", "3", "[]"]) expect(parseBonusGuide(raw)).toEqual(FRESH_GUIDE)
    })

    it("reads back what it wrote", () => {
        const guide = afterWin(afterUse(afterUse(FRESH_GUIDE, "bomb"), "shields"), "encloseClicks")

        expect(parseBonusGuide(JSON.stringify(guide))).toEqual(guide)
    })

    it("keeps only the kinds it knows and the counts that make sense", () => {
        const raw = JSON.stringify({
            won: ["bomb", "tripleClicks", 4],
            uses: {spreadClicks: 2, shields: -1, refill: 1.5, encloseClicks: 99, tripleClicks: 1},
        })

        expect(parseBonusGuide(raw)).toEqual({won: ["bomb"], uses: {spreadClicks: 2, encloseClicks: LEARNING_USES}})
    })
})

describe("the guide", () => {
    it("calls a kind new until its first use", () => {
        const guide = afterUse(FRESH_GUIDE, "bomb")

        expect(isNew(FRESH_GUIDE, "bomb")).toBe(true)
        expect(isNew(guide, "bomb")).toBe(false)
        expect(isNew(guide, "shields")).toBe(true)
    })

    it("explains a kind for its first uses, then stops", () => {
        let guide = FRESH_GUIDE
        for (let use = 0; use < LEARNING_USES; use++) {
            expect(isLearning(guide, "encloseClicks")).toBe(true)
            guide = afterUse(guide, "encloseClicks")
        }

        expect(isLearning(guide, "encloseClicks")).toBe(false)
        expect(afterUse(guide, "encloseClicks")).toBe(guide)
        expect(usesOf(guide, "encloseClicks")).toBe(LEARNING_USES)
    })

    it("knows the first win of each kind", () => {
        const guide = afterWin(FRESH_GUIDE, "shields")

        expect(isFirstWin(FRESH_GUIDE, "shields")).toBe(true)
        expect(isFirstWin(guide, "shields")).toBe(false)
        expect(isFirstWin(guide, "bomb")).toBe(true)
        expect(afterWin(guide, "shields")).toBe(guide)
    })
})
