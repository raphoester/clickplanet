import {beforeEach, describe, expect, it, vi} from "vitest"
import * as THREE from "three"
import {TileField} from "./tileField.ts"
import {regions} from "./atlas.ts"
import {resetWarnOnce} from "../../domain/warnOnce.ts"
import type {PointGeometryData} from "./coordinatesBinary.ts"

const SIZE = 32

function field(size = SIZE) {
    const data: PointGeometryData = {
        positions: new Float32Array(size * 3),
        uvs: new Float32Array(size * 2),
        size,
    }
    return new TileField({zoom: {value: 1}}, data)
}

const attr = (f: TileField, name: string) =>
    f.displayPoints.geometry.getAttribute(name) as THREE.BufferAttribute

const hoverValues = (f: TileField) => Array.from(attr(f, "hover").array as Float32Array)

beforeEach(() => {
    resetWarnOnce()
    vi.spyOn(console, "warn").mockImplementation(() => {})
})

describe("setOwners", () => {
    it("writes the country's atlas region at the tile's offset", () => {
        const f = field()
        const fr = regions.get("fr")!

        f.setOwners([{tile: 3, country: "fr"}])

        const values = attr(f, "regionVector").array as Float32Array
        expect(Array.from(values.slice(8, 12))).toEqual([fr.x, fr.y, fr.width, fr.height])
    })

    it("leaves every other tile untouched", () => {
        const f = field()
        f.setOwners([{tile: 3, country: "fr"}])

        const values = attr(f, "regionVector").array as Float32Array
        expect(Array.from(values.slice(0, 8))).toEqual([0, 0, 0, 0, 0, 0, 0, 0])
        expect(Array.from(values.slice(12, 16))).toEqual([0, 0, 0, 0])
    })

    it("starts every tile unowned, which the shader draws blank", () => {
        const values = attr(field(), "regionVector").array as Float32Array
        expect(values.every(v => v === 0)).toBe(true)
    })

    it("leaves a tile blank and warns once for a country with no region", () => {
        const f = field()
        f.setOwners([{tile: 1, country: "zz"}, {tile: 2, country: "zz"}])

        const values = attr(f, "regionVector").array as Float32Array
        expect(Array.from(values.slice(0, 8))).toEqual([0, 0, 0, 0, 0, 0, 0, 0])
        expect(console.warn).toHaveBeenCalledTimes(1)
    })

    it("blanks the tile again when a rolled-back click leaves it unowned", () => {
        const f = field()
        f.setOwners([{tile: 3, country: "fr"}])

        f.setOwners([{tile: 3, country: undefined}])

        const values = attr(f, "regionVector").array as Float32Array
        expect(Array.from(values.slice(8, 12))).toEqual([0, 0, 0, 0])
        expect(console.warn).not.toHaveBeenCalled()
    })

    it("uploads the tile a rollback blanked", () => {
        const f = field()
        f.setOwners([{tile: 2, country: "fr"}])
        const region = attr(f, "regionVector")
        region.clearUpdateRanges()

        f.setOwners([{tile: 2, country: undefined}])

        expect(region.updateRanges).toEqual([{start: 4, count: 4}])
    })

    it("does not touch the buffer when nothing changed", () => {
        const f = field()
        const region = attr(f, "regionVector")
        region.needsUpdate = false

        f.setOwners([])
        expect(region.updateRanges).toEqual([])
    })

    it("keeps the same attribute and backing array across changes", () => {
        const f = field()
        const region = attr(f, "regionVector")
        const array = region.array

        f.setOwners([{tile: 1, country: "fr"}])
        f.setOwners([{tile: 2, country: "jp"}])

        expect(attr(f, "regionVector")).toBe(region)
        expect(attr(f, "regionVector").array).toBe(array)
    })

    it("uploads only the tiles that changed", () => {
        const f = field()
        const region = attr(f, "regionVector")
        region.clearUpdateRanges()
        const version = region.version

        f.setOwners([{tile: 2, country: "fr"}])

        expect(region.updateRanges).toEqual([{start: 4, count: 4}])
        expect(region.version).toBe(version + 1)
    })

    it("collapses a large batch into a single spanning range", () => {
        const f = field(1000)
        const region = attr(f, "regionVector")
        region.clearUpdateRanges()

        const changes = Array.from({length: 200}, (_, i) => ({tile: i + 1, country: "fr"}))
        f.setOwners(changes)

        expect(region.updateRanges).toEqual([{start: 0, count: 800}])
    })
})

describe("setHover", () => {
    it("marks the hovered tile", () => {
        const f = field()
        f.setHover(4)
        expect(hoverValues(f)[3]).toBe(1)
    })

    it("clears the previously hovered tile", () => {
        const f = field()
        f.setHover(4)
        f.setHover(9)

        expect(hoverValues(f)[3]).toBe(0)
        expect(hoverValues(f)[8]).toBe(1)
    })

    it("clears everything when nothing is hovered", () => {
        const f = field()
        f.setHover(4)
        f.setHover(undefined)
        expect(hoverValues(f).every(v => v === 0)).toBe(true)
    })

    it("highlights exactly one tile at a time", () => {
        const f = field()
        for (const tile of [1, 7, 20, 32]) f.setHover(tile)
        expect(hoverValues(f).filter(v => v === 1)).toEqual([1])
    })

    it("does no work when the hovered tile repeats", () => {
        const f = field()
        f.setHover(4)
        const hover = attr(f, "hover")
        hover.clearUpdateRanges()

        f.setHover(4)
        expect(hover.updateRanges).toEqual([])
    })

    it("keeps the same attribute and backing array across moves", () => {
        const f = field()
        const hover = attr(f, "hover")
        const array = hover.array

        for (let tile = 1; tile <= SIZE; tile++) f.setHover(tile)

        expect(attr(f, "hover")).toBe(hover)
        expect(attr(f, "hover").array).toBe(array)
    })

    it("uploads only the two tiles involved", () => {
        const f = field()
        f.setHover(4)
        const hover = attr(f, "hover")
        hover.clearUpdateRanges()

        f.setHover(9)

        expect(hover.updateRanges).toEqual([{start: 3, count: 1}, {start: 8, count: 1}])
    })
})

describe("geometry", () => {
    it("gives the picking points one colour per tile, starting at id 1", () => {
        const f = field()
        const colors = f.pickingPoints.geometry.getAttribute("color").array as Float32Array

        expect(colors[0]).toBe(0)
        expect(colors[1]).toBe(0)
        expect(colors[2]).toBeCloseTo(1 / 255, 6)
        expect(colors.length).toBe(SIZE * 3)
    })

    it("shares one position attribute between both point clouds", () => {
        const f = field()
        expect(f.pickingPoints.geometry.getAttribute("position"))
            .toBe(f.displayPoints.geometry.getAttribute("position"))
    })

    it("does not upload a uv attribute", () => {
        const f = field()
        expect(f.displayPoints.geometry.getAttribute("uv")).toBeUndefined()
        expect(f.pickingPoints.geometry.getAttribute("uv")).toBeUndefined()
    })
})
