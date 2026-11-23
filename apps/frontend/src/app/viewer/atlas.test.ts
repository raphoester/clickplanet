import {fileURLToPath} from "node:url"
import sharp from "sharp"
import {describe, expect, it} from "vitest"
import {BLANK_NUMBER, BLANK_REGION, blankOf, flagNumberOf, flagNumbers, regionOf, regions} from "./atlas.ts"
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts"
import {Countries} from "../../domain/countries.ts"

describe("flag atlas", () => {
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

describe("the blank flag", () => {
    it("is a flag-shaped rectangle of the atlas that comes near no picture", () => {
        const {x, y, width, height} = BLANK_REGION
        expect(width).toBeGreaterThan(height)
        expect(x + width).toBeLessThanOrEqual(ATLAS_SIZE.width)
        expect(y + height).toBeLessThanOrEqual(ATLAS_SIZE.height)
        const near = Array.from(regions.entries())
            .filter(([, r]) => x < r.x + r.width + 4 && r.x < x + width + 4 && y < r.y + r.height + 4 && r.y < y + height + 4)
            .map(([code]) => code)
        expect(near).toEqual([])
    })

    it("is transparent in the atlas, which the shaders draw white", async () => {
        const file = fileURLToPath(new URL(`../../..${ATLAS_URL}`, import.meta.url))
        const {data} = await sharp(file)
            .extract({left: BLANK_REGION.x, top: BLANK_REGION.y, width: BLANK_REGION.width, height: BLANK_REGION.height})
            .ensureAlpha().raw().toBuffer({resolveWithObject: true})
        const alphas = new Set(Array.from({length: data.length / 4}, (_, i) => data[i * 4 + 3]))
        expect([...alphas]).toEqual([0])
    })

    it("is what a flag with no picture wears, and no flag of the build", () => {
        expect(regionOf("r-k3p9x2ma0b")).toBe(BLANK_REGION)
        expect(flagNumberOf("r-k3p9x2ma0b")).toBe(BLANK_NUMBER)
        expect(regionOf("fr")).toBe(regions.get("fr"))
        expect(flagNumberOf("fr")).toBe(flagNumbers.get("fr"))
        expect([...flagNumbers.values()]).not.toContain(BLANK_NUMBER)
    })

    it("refuses an atlas with no room left", () => {
        const full = [{x: 0, y: 0, width: 100, height: 50}, {x: 100, y: 0, width: 100, height: 50}]
        expect(() => blankOf(full, {width: 200, height: 50})).toThrow()
    })
})
