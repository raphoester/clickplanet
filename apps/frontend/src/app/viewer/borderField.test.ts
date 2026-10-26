import {describe, expect, it} from "vitest"
import {BorderField, countryOfTile, type BorderData} from "./borderField.ts"
import {regions} from "./atlas.ts"

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

const rows = (field: BorderField) => field.landmassData.image.data as unknown as Float32Array
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
        expect(opacityOf(field)).toBe(0)
    })

    it("ignores a landmass the table has no row for", () => {
        const field = new BorderField(world(8))

        field.setRows([0, 2, 99], () => ({holder: "fr", held: 8}))

        expect(field.holderOf(1)).toBeUndefined()
    })
})

describe("what a landmass has to earn before it paints", () => {
    it("stays bare Earth below the tile floor, however completely it is held", () => {
        const field = new BorderField(world(3), undefined, undefined, 4)

        paint(field, {fr: 3})

        expect(field.holderOf(1)).toBeUndefined()
        expect(regionOf(field)).toEqual([0, 0, 0, 0])
    })

    it("stays bare Earth below the share floor, rather than showing a ghost", () => {
        const field = new BorderField(world(8), 0.5)

        paint(field, {fr: 3})

        expect(field.holderOf(1)).toBeUndefined()
        expect(opacityOf(field)).toBe(0)
    })

    it("paints once the leader crosses the floor", () => {
        const field = new BorderField(world(8), 0.5)

        paint(field, {fr: 5})

        expect(field.holderOf(1)).toBe("fr")
        expect(opacityOf(field)).toBeCloseTo(5 / 8)
    })
})

describe("opacity", () => {
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
})

describe("the landmass table", () => {
    it("is not re-uploaded when nothing about a landmass changed", () => {
        const field = new BorderField(world(8))
        paint(field, {fr: 5})

        const uploaded = field.landmassData.version
        paint(field, {fr: 5})
        expect(field.landmassData.version).toBe(uploaded)

        paint(field, {de: 5})
        expect(field.landmassData.version).toBeGreaterThan(uploaded)
    })
})
