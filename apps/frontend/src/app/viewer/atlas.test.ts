import {describe, expect, it} from "vitest"
import {regions} from "./atlas.ts"
import {Countries} from "../../domain/countries.ts"

describe("flag atlas", () => {
    it("exposes a region per entry in the atlas manifest", () => {
        expect(regions.size).toBeGreaterThan(0)
    })

    it("covers every country the picker can select", () => {
        const missing = Array.from(Countries.keys()).filter(code => !regions.has(code))
        expect(missing).toEqual([])
    })

    it("describes every region as a non-empty rectangle", () => {
        const degenerate = Array.from(regions.entries())
            .filter(([, r]) => !(r.width > 0 && r.height > 0 && r.x >= 0 && r.y >= 0))
            .map(([code]) => code)
        expect(degenerate).toEqual([])
    })
})
