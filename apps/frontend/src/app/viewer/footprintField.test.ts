import {describe, expect, it} from "vitest"
import {flagCodeOf, regions} from "./atlas.ts"
import {FootprintField, FOOTPRINT_COLUMNS, MOST_FOOTPRINTS, type PaintedGrounds} from "./footprintField.ts"
import type {Divisions} from "./pointSize.ts"
import {flagNeedOf, type ScreenView} from "./slotNeed.ts"

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

    describe("flagNeeds", () => {
        const view: ScreenView = {
            toward: {x: 0, y: 0, z: 1},
            right: {x: 1, y: 0, z: 0},
            up: {x: 0, y: 1, z: 0},
            origin: {x: 0, y: 0},
            pixelsPerRadian: 1000,
            width: 1000,
            height: 800,
        }
        const leader = flagCodeOf(1)!
        const other = flagCodeOf(2)!
        const sizes = {countries: [0, 4], provinces: [0, 2, 2]}
        const grounds = (divisions: Divisions, holder: string | undefined): PaintedGrounds => ({
            holderOf: () => holder,
            sizeOf: (ground) => sizes[divisions][ground],
        })
        const LEAST = 0.05

        it("asks for each flag that does not lead a ground in view, as wide as its footprint's frame on screen", () => {
            const needs = make().flagNeeds("countries", view, Float32Array.of(0, 1000), grounds("countries", leader), LEAST)
            const {width, height} = regions.get(other)!
            const aspect = width / height
            const centre = {x: 0.3 / Math.hypot(0.3, 1), y: 0, z: 1 / Math.hypot(0.3, 1)}
            expect([...needs.keys()]).toEqual([other])
            expect(needs.get(other)!.need).toBeCloseTo(flagNeedOf(centre, LEAST * aspect, LEAST, aspect, view))
            expect(needs.get(other)!.area).toBeCloseTo(250)
        })

        it("asks for every flag on a ground nobody leads, each with its share of the land", () => {
            const needs = make().flagNeeds("countries", view, Float32Array.of(0, 1000), grounds("countries", undefined), LEAST)
            expect(needs.get(leader)!.area).toBeCloseTo(750)
            expect(needs.get(other)!.area).toBeCloseTo(250)
        })

        it("asks for nothing on a ground off screen, and reads the provinces' footprints past the provinces' line", () => {
            const field = make()
            expect(field.flagNeeds("countries", view, Float32Array.of(0, 0), grounds("countries", undefined), LEAST).size).toBe(0)
            const needs = field.flagNeeds("provinces", view, Float32Array.of(0, 0, 600), grounds("provinces", leader), LEAST)
            expect([...needs.keys()]).toEqual([other])
            expect(needs.get(other)!.area).toBeCloseTo(300)
        })
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
