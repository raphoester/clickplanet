import {fileURLToPath} from "node:url"
import sharp from "sharp"
import {describe, expect, it} from "vitest"
import {
    aspectOf, BLANK_NUMBER, BLANK_REGION, blankOf, FIRST_SOLD, FLAG_NUMBERS, flagCodeOf, flagNumberOf, flagNumbers, knowPicture,
    regionOf, regionOfNumber, regions, showPicture,
} from "./atlas.ts"
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

    it("is what a sold flag wears until its picture is shown, and no flag of the build", () => {
        expect(regionOf("r-blank00000")).toBe(BLANK_REGION)
        expect(regionOf("fr")).toBe(regions.get("fr"))
        expect(flagNumberOf("fr")).toBe(flagNumbers.get("fr"))
        expect([...flagNumbers.values()]).not.toContain(BLANK_NUMBER)
        expect(flagNumberOf("")).toBe(BLANK_NUMBER)
    })
})

describe("a sold flag", () => {
    it("takes a number of its own after the blank, the same each time, and its code back from it", () => {
        const number = flagNumberOf("r-numbered00")
        expect(number).toBeGreaterThanOrEqual(FIRST_SOLD)
        expect(number).toBeLessThan(FLAG_NUMBERS)
        expect(flagNumberOf("r-numbered00")).toBe(number)
        expect(flagNumberOf("r-numbered01")).toBe(number + 1)
        expect(flagCodeOf(number)).toBe("r-numbered00")
    })

    it("wears its cell once its picture is shown, and the blank again once it is not", () => {
        const number = flagNumberOf("r-shown00000")
        const cell = {x: 100, y: 1248, width: 100, height: 67}
        showPicture(number, cell)
        expect(regionOf("r-shown00000")).toBe(cell)
        expect(regionOfNumber(number)).toBe(cell)
        showPicture(number, undefined)
        expect(regionOf("r-shown00000")).toBe(BLANK_REGION)
    })

    it("has its picture's shape, as a built flag has its sprite's", () => {
        knowPicture("r-shaped0000", {url: "x.png", width: 768, height: 384, stretch: false, focusX: 0.5, focusY: 0.5})
        expect(aspectOf("r-shaped0000")).toBe(2)
        expect(aspectOf("fr")).toBeCloseTo(regions.get("fr")!.width / regions.get("fr")!.height)
        expect(aspectOf("r-unknown000")).toBeUndefined()
    })

    it("refuses an atlas with no room left", () => {
        const full = [{x: 0, y: 0, width: 100, height: 50}, {x: 100, y: 0, width: 100, height: 50}]
        expect(() => blankOf(full, {width: 200, height: 50})).toThrow()
    })
})
