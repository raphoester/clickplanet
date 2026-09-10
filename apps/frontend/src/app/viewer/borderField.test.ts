import {describe, expect, it} from "vitest"
import {BorderField, type BorderData} from "./borderField.ts"
import {regions} from "./atlas.ts"

const TEXELS = 4

// A world of `total` tiles, all in one landmass, plus `spare` tiles in none.
function world(total: number, spare = 0): BorderData {
    const codes = ["", "xx"]
    const assignment = new Uint16Array(total + spare)
    assignment.fill(1, 0, total)

    const frames = new Float32Array(codes.length * 5)
    frames[5] = 0
    frames[6] = 0
    frames[7] = 1      // centre, somewhere on the sphere
    frames[8] = 0.1    // how far its tiles reach, east and north
    frames[9] = 0.1

    const totals = new Uint32Array(codes.length)
    totals[1] = total
    return {codes, assignment, frames, totals}
}

const rows = (field: BorderField) => field.landmassData.image.data as unknown as Float32Array
const opacityOf = (field: BorderField, piece = 1) => rows(field)[piece * TEXELS * 4 + 12]
const regionOf = (field: BorderField, piece = 1) =>
    Array.from(rows(field).slice(piece * TEXELS * 4 + 8, piece * TEXELS * 4 + 12))

const claims = (country: string | undefined, from: number, to: number) =>
    Array.from({length: to - from + 1}, (_, i) => ({tile: from + i, country}))

describe("who a landmass flies", () => {
    it("is whoever holds the most of it, not whoever arrived last", () => {
        const field = new BorderField(world(8), 8)

        field.apply(claims("fr", 1, 5))
        field.apply(claims("de", 6, 8))

        expect(field.holderOf(1)).toBe("fr")
        expect(regionOf(field)).toEqual(Object.values(regions.get("fr")!))
    })

    it("changes hands when the balance actually tips", () => {
        const field = new BorderField(world(8), 8)
        field.apply(claims("fr", 1, 5))
        expect(field.holderOf(1)).toBe("fr")

        // Three of France's five go over, so Germany leads 6-2.
        field.apply(claims("de", 1, 3))
        field.apply(claims("de", 6, 8))

        expect(field.holderOf(1)).toBe("de")
        expect(opacityOf(field)).toBeCloseTo(6 / 8)
    })

    it("lets go of a tile that goes back to unowned", () => {
        const field = new BorderField(world(8), 8)
        field.apply(claims("fr", 1, 5))
        field.apply(claims("de", 6, 8))

        field.apply(claims(undefined, 1, 4))

        expect(field.holderOf(1)).toBe("de")
        expect(opacityOf(field)).toBeCloseTo(3 / 8)
    })

    it("ignores a tile that sits in no country at all", () => {
        const field = new BorderField(world(8, 4), 12)

        field.apply(claims("fr", 9, 12))

        expect(field.holderOf(1)).toBeUndefined()
        expect(opacityOf(field)).toBe(0)
    })
})

describe("what a landmass has to earn before it paints", () => {
    it("stays bare Earth below the tile floor, however completely it is held", () => {
        const field = new BorderField(world(3), 3, undefined, undefined, 4)

        field.apply(claims("fr", 1, 3))

        expect(field.holderOf(1)).toBeUndefined()
        // A zero-width region is what the shader reads as "nobody holds this".
        expect(regionOf(field)).toEqual([0, 0, 0, 0])
    })

    it("stays bare Earth below the share floor, rather than showing a ghost", () => {
        const field = new BorderField(world(8), 8, 0.5)

        field.apply(claims("fr", 1, 3))

        expect(field.holderOf(1)).toBeUndefined()
        expect(opacityOf(field)).toBe(0)
    })

    it("paints once the leader crosses the floor", () => {
        const field = new BorderField(world(8), 8, 0.5)

        field.apply(claims("fr", 1, 5))

        expect(field.holderOf(1)).toBe("fr")
        expect(opacityOf(field)).toBeCloseTo(5 / 8)
    })
})

describe("opacity", () => {
    it("is the leader's share of the landmass", () => {
        for (const [held, total] of [[1, 8], [4, 8], [8, 8], [37, 100]]) {
            const field = new BorderField(world(total), total)
            field.apply(claims("fr", 1, held))
            expect(opacityOf(field), `${held}/${total}`).toBeCloseTo(held / total)
        }
    })

    // The regression that made a country brighter the further in you zoomed.
    // Zoomed in, a holder's share is already on screen as the fraction of discs
    // wearing their flag, so the tiles show `share * 0.7` of ink. The painted
    // flag shows `opacity * 0.94`. Whenever the flag is the fainter of the two,
    // the crossfade between them ramps up instead of down.
    it("is never fainter than the tiles the flag hands over to", () => {
        for (const held of [8, 20, 37, 50, 75, 90, 100]) {
            const field = new BorderField(world(100), 100)
            field.apply(claims("fr", 1, held))

            const share = held / 100
            expect(opacityOf(field) * 0.94, `holding ${held}%`).toBeGreaterThanOrEqual(share * 0.7)
        }
    })
})

describe("the landmass table", () => {
    it("carries how far the landmass reaches, whoever holds it", () => {
        const field = new BorderField(world(8), 8)
        const reach = () => rows(field)[TEXELS * 4 + 13]

        expect(reach()).toBeCloseTo(0.1)
        field.apply(claims("fr", 1, 8))
        expect(reach()).toBeCloseTo(0.1)
    })

    // `needsUpdate` is write-only on a texture; the upload it schedules shows up
    // as a bumped version.
    it("is not re-uploaded when nothing about a landmass changed", () => {
        const field = new BorderField(world(8), 8)
        field.apply(claims("fr", 1, 5))

        const uploaded = field.landmassData.version
        field.apply(claims("fr", 1, 5))
        expect(field.landmassData.version).toBe(uploaded)

        field.apply(claims("de", 1, 5))
        expect(field.landmassData.version).toBeGreaterThan(uploaded)
    })
})
