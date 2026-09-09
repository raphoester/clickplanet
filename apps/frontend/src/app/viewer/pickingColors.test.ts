import {describe, expect, it} from "vitest"
import {colorToInteger, integerToColor} from "./pickingColors.ts"

describe("picking colors", () => {
    it("packs an id into the three colour channels", () => {
        expect(integerToColor(1)).toEqual([0, 0, 1])
        expect(integerToColor(255)).toEqual([0, 0, 255])
        expect(integerToColor(256)).toEqual([0, 1, 0])
        expect(integerToColor(65_536)).toEqual([1, 0, 0])
    })

    it("unpacks the channels back into the id", () => {
        expect(colorToInteger([0, 0, 1])).toBe(1)
        expect(colorToInteger([0, 1, 0])).toBe(256)
        expect(colorToInteger([1, 0, 0])).toBe(65_536)
    })

    it("round-trips every id in the range the tile map uses", () => {
        const broken: number[] = []
        for (let id = 1; id <= 300_000; id++) {
            if (colorToInteger(integerToColor(id)) !== id) broken.push(id)
        }
        expect(broken).toEqual([])
    })

    it("round-trips the whole 24-bit range the encoding can address", () => {
        for (const id of [1, 0xff, 0x100, 0xffff, 0x10000, 0xfffffe, 0xffffff]) {
            expect(colorToInteger(integerToColor(id))).toBe(id)
        }
    })

    it("reserves the two sentinel colours outside the usable id range", () => {
        expect(colorToInteger([0, 0, 0])).toBe(0)
        expect(colorToInteger([255, 255, 255])).toBe(0xffffff)
    })
})
