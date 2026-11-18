import {describe, expect, it} from "vitest"
import {BorderField, countryOfTile, leastPaintedTiles, type BorderData} from "./borderField.ts"
import {EARTH_1_SCALE, scaleOf} from "../../domain/boardScale.ts"
import {FIXTURE_1} from "../../testdata/boards.ts"
import {flagNumbers, regions} from "./atlas.ts"
import {LandmassSight, type ScreenView} from "./slotNeed.ts"

const TEXELS = 4

function world(total: number, spare = 0): BorderData {
    const codes = ["", "xx"]
    const assignment = new Uint16Array(total + spare)
    assignment.fill(1, 0, total)

    const frames = new Float32Array(codes.length * 5)
    frames[5] = 0
    frames[6] = 0
    frames[7] = 1
    frames[8] = 0.1
    frames[9] = 0.1

    const totals = new Uint32Array(codes.length)
    totals[1] = total
    return {codes, assignment, frames, totals}
}

const rows = (field: BorderField) => field.texture.image.data as unknown as Float32Array
const opacityOf = (field: BorderField, piece = 1) => rows(field)[piece * TEXELS * 4 + 12]
const regionOf = (field: BorderField, piece = 1) =>
    Array.from(rows(field).slice(piece * TEXELS * 4 + 8, piece * TEXELS * 4 + 12))

// Paints landmass 1 from its row: the tiles each flag holds of it.
const paint = (field: BorderField, row: Record<string, number>) => field.setRows([1], () => {
    const [holder, held] = Object.entries(row).sort((a, b) => b[1] - a[1])[0] ?? []
    return holder === undefined ? undefined : {holder, held}
})

describe("countryOfTile", () => {
    it("names the country a tile lies in, and nothing outside every country", () => {
        const data = world(3, 1)
        expect(countryOfTile(data, 1)).toBe("xx")
        expect(countryOfTile(data, 4)).toBeUndefined()
        expect(countryOfTile(data, 99)).toBeUndefined()
    })
})

describe("who a landmass flies", () => {
    it("is the leader of its row", () => {
        const field = new BorderField(world(8))

        paint(field, {fr: 5, de: 3})

        expect(field.holderOf(1)).toBe("fr")
        expect(regionOf(field)).toEqual(Object.values(regions.get("fr")!))
    })

    it("changes hands when the balance actually tips", () => {
        const field = new BorderField(world(8))
        paint(field, {fr: 5})
        expect(field.holderOf(1)).toBe("fr")

        paint(field, {fr: 2, de: 6})

        expect(field.holderOf(1)).toBe("de")
        expect(regionOf(field)).toEqual(Object.values(regions.get("de")!))
        expect(opacityOf(field)).toBeCloseTo(6 / 8)
    })

    it("lets go of the tiles nobody holds any more", () => {
        const field = new BorderField(world(8))
        paint(field, {fr: 5, de: 3})

        paint(field, {fr: 1, de: 3})

        expect(field.holderOf(1)).toBe("de")
        expect(opacityOf(field)).toBeCloseTo(3 / 8)
    })

    it("is nobody once nobody holds any of it", () => {
        const field = new BorderField(world(8))
        paint(field, {fr: 5})

        paint(field, {})

        expect(field.holderOf(1)).toBeUndefined()
        expect(regionOf(field)).toEqual([0, 0, 0, 0])
    })

    it("ignores a landmass the table has no row for", () => {
        const field = new BorderField(world(8))

        field.setRows([0, 2, 99], () => ({holder: "fr", held: 8}))

        expect(field.holderOf(1)).toBeUndefined()
    })
})

describe("what a flag has to earn before it leads a landmass", () => {
    it("leads nothing below the tile floor, however completely it holds it", () => {
        const field = new BorderField(world(3), undefined, 4)

        paint(field, {fr: 3})

        expect(field.holderOf(1)).toBeUndefined()
        expect(regionOf(field)).toEqual([0, 0, 0, 0])
    })

    it("leads nothing below the share floor", () => {
        const field = new BorderField(world(8), 0.5)

        paint(field, {fr: 3})

        expect(field.holderOf(1)).toBeUndefined()
        expect(regionOf(field)).toEqual([0, 0, 0, 0])
    })

    it("leads once it crosses the floor", () => {
        const field = new BorderField(world(8), 0.5)

        paint(field, {fr: 5})

        expect(field.holderOf(1)).toBe("fr")
        expect(regionOf(field)).toEqual(Object.values(regions.get("fr")!))
    })
})

describe("the big flag's opacity", () => {
    it("is the leader's share of the landmass", () => {
        for (const [held, total] of [[1, 8], [4, 8], [8, 8], [37, 100]]) {
            const field = new BorderField(world(total))
            paint(field, {fr: held})
            expect(opacityOf(field), `${held}/${total}`).toBeCloseTo(held / total)
        }
    })

    it("is never fainter than the tiles the flag hands over to", () => {
        for (const held of [8, 20, 37, 50, 75, 90, 100]) {
            const field = new BorderField(world(100))
            paint(field, {fr: held})

            const share = held / 100
            expect(opacityOf(field) * 0.94, `holding ${held}%`).toBeGreaterThanOrEqual(share * 0.7)
        }
    })

    it("is nothing below the share floor", () => {
        const field = new BorderField(world(8), 0.5)
        paint(field, {fr: 3})
        expect(opacityOf(field)).toBe(0)
    })
})

describe("the flags that do not lead", () => {
    it("find each landmass's reach in its row, led or not", () => {
        const field = new BorderField(world(8))

        expect(Array.from(rows(field).slice(TEXELS * 4 + 4, TEXELS * 4 + 6))).toEqual([0.1, 0.1].map(Math.fround))
    })
})

describe("the flags drawn large", () => {
    const facing = (z: number): ScreenView => ({
        toward: {x: 0, y: 0, z},
        right: {x: z, y: 0, z: 0},
        up: {x: 0, y: 1, z: 0},
        origin: {x: 0, y: 0},
        pixelsPerRadian: 1000,
        width: 1000,
        height: 1000,
    })
    const sight = new LandmassSight((landmass) => (landmass === 1 ? [1, 2] : []), 2, new Float32Array([0, 0, 1, 0, 0, 1]), 1e-4)

    it("need each painted flag on screen, and nothing of ground nobody paints", () => {
        const field = new BorderField(world(8))
        expect(field.flagNeeds(facing(1), sight.areasOnScreen(facing(1))).size).toBe(0)

        paint(field, {fr: 5})

        expect(field.flagNeeds(facing(1), sight.areasOnScreen(facing(1)))).toEqual(new Map([["fr", {
            need: expect.closeTo(298.5, 1),
            area: expect.closeTo(200, 3),
            aspect: 100 / 67,
            spriteWidth: 100,
        }]]))
    })

    it("need nothing of a landmass behind the earth", () => {
        const field = new BorderField(world(8))
        paint(field, {fr: 5})

        expect(field.flagNeeds(facing(-1), sight.areasOnScreen(facing(-1))).get("fr")).toMatchObject({need: 0, area: 0})
    })

    it("keep the leader's number in its row, for the slot the territory layer reads", () => {
        const field = new BorderField(world(8))
        paint(field, {fr: 5})
        expect(rows(field)[TEXELS * 4 + 13]).toBe(flagNumbers.get("fr"))

        paint(field, {})
        expect(rows(field)[TEXELS * 4 + 13]).toBe(0)
    })
})

describe("the landmass table", () => {
    it("is not re-uploaded when nothing about a landmass changed", () => {
        const field = new BorderField(world(8))
        paint(field, {fr: 5})

        const uploaded = field.texture.version
        paint(field, {fr: 5})
        expect(field.texture.version).toBe(uploaded)

        paint(field, {de: 5})
        expect(field.texture.version).toBeGreaterThan(uploaded)
    })
})

describe("the least landmass a flag is painted in", () => {
    it("is 4 tiles on earth-1, and as much ground on a coarser board", () => {
        expect(leastPaintedTiles(EARTH_1_SCALE)).toBe(4)
        expect(leastPaintedTiles(scaleOf(FIXTURE_1))).toBe(1)
    })
})
