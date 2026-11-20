import {describe, expect, it} from "vitest"
import {FootprintField, FOOTPRINT_COLUMNS, MOST_FOOTPRINTS} from "./footprintField.ts"

const TEXEL = 4
const ROW = FOOTPRINT_COLUMNS * 2 * TEXEL
const LANDMASS_ROWS = (MOST_FOOTPRINTS / FOOTPRINT_COLUMNS) * ROW

function frameAt(field: FootprintField, id: number, landmasses = false): number[] {
    const values = field.frames.image.data as unknown as Float32Array
    const at = (landmasses ? LANDMASS_ROWS : 0) + id * 2 * TEXEL
    return Array.from(values.subarray(at, at + 2 * TEXEL))
}

describe("FootprintField", () => {
    // Four tiles on the equator; tile i at texel 3 - i. Provinces 1, 1, 2, 2, both on landmass 1.
    const positions = Float32Array.of(0, 0, 1, 0.1, 0, 1, 0.2, 0, 1, 0.3, 0, 1)
    const texels = Uint32Array.of(3, 2, 1, 0)
    const make = () => {
        const field = new FootprintField(positions, texels, 4)
        field.setFlags([0, 1, 2, 3], [1, 1, 1, 2])
        field.setGrounds(Uint16Array.of(1, 1, 2, 2), Uint16Array.of(0, 1, 1), 3, 2)
        return field
    }

    it("counts the flags set before the grounds, and writes each tile's province footprint at its texel", () => {
        const field = make()
        const ids = field.tileData.image.data as unknown as Uint16Array
        expect(ids[3]).toBe(ids[2])
        expect(ids[1]).not.toBe(ids[3])
        expect(ids[0]).not.toBe(ids[1])
        expect(frameAt(field, ids[3])[7]).toBe(2)
    })

    it("names each province footprint's landmass footprint, which counts the flag's tiles over both provinces", () => {
        const field = make()
        const ids = field.tileData.image.data as unknown as Uint16Array
        const parent = frameAt(field, ids[3])[5]
        expect(frameAt(field, ids[1])[5]).toBe(parent)
        expect(frameAt(field, ids[0])[5]).not.toBe(parent)
        expect(frameAt(field, parent, true)[7]).toBe(3)
    })

    it("clears a footprint's frame once its flag holds none of its tiles", () => {
        const field = make()
        const ids = field.tileData.image.data as unknown as Uint16Array
        const lone = ids[0]
        field.setFlags([3], [1])
        field.upload({copyTextureToTexture: () => undefined} as never)
        expect(frameAt(field, lone)).toEqual([0, 0, 0, 0, 0, 0, 0, 0])
        expect(ids[0]).toBe(ids[1])
    })
})
