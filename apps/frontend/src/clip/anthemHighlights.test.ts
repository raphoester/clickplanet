import {describe, expect, it} from "vitest"
import {ANTHEMS} from "../app/anthem/anthemsAsset.ts"
import {HIGHLIGHTS} from "./anthemHighlightsAsset.ts"
import {CLIP_ANTHEMS} from "./clipAnthemsAsset.ts"

describe("the highlights of the anthems", () => {
    it("are measured for every anthem a clip can play", () => {
        const urls = new Set([...Object.values(ANTHEMS), ...Object.values(CLIP_ANTHEMS)].map((anthem) => anthem.url))

        expect([...urls].filter((url) => !(url in HIGHLIGHTS))).toEqual([])
    })

    it("are found in every anthem", () => {
        expect(Object.entries(HIGHLIGHTS).filter(([, anthem]) => anthem.at.length === 0).map(([url]) => url)).toEqual([])
    })
})
