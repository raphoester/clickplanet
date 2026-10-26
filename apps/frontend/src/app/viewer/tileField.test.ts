import {beforeEach, describe, expect, it, vi} from "vitest"
import * as THREE from "three"
import {TileField} from "./tileField.ts"
import {regions} from "./atlas.ts"
import {resetWarnOnce} from "../../domain/warnOnce.ts"
import type {PointGeometryData} from "./coordinatesBinary.ts"

const SIZE = 32

type Slot = {name: string, offset: number, width: number}

type Layout = {compact: boolean, flag: Slot, landmass: Slot, hover: Slot, shield: Slot, pulse: Slot}

const PLAIN: Layout = {
    compact: false,
    flag: {name: "regionVector", offset: 0, width: 4},
    landmass: {name: "landmassIndex", offset: 0, width: 1},
    hover: {name: "hover", offset: 0, width: 1},
    shield: {name: "shield", offset: 0, width: 1},
    pulse: {name: "pulse", offset: 0, width: 1},
}

const COMPACT: Layout = {
    compact: true,
    flag: {name: "ids", offset: 0, width: 1},
    landmass: {name: "ids", offset: 1, width: 1},
    hover: {name: "marks", offset: 0, width: 1},
    pulse: {name: "marks", offset: 1, width: 1},
    shield: {name: "marks", offset: 2, width: 1},
}

function fieldOf(compact: boolean, size = SIZE) {
    const data: PointGeometryData = {
        positions: new Float32Array(size * 3),
        uvs: new Float32Array(size * 2),
        size,
    }
    return new TileField({zoom: {value: 1}}, {pointSize: {value: 1}}, data, {tiles: false, compact})
}

const attr = (f: TileField, name: string) =>
    f.displayPoints.geometry.getAttribute(name) as THREE.BufferAttribute

const ZERO = [0, 0, 0, 0]

beforeEach(() => {
    resetWarnOnce()
    vi.spyOn(console, "warn").mockImplementation(() => {})
})

describe.each([["plain", PLAIN], ["compact", COMPACT]] as const)("the %s layout", (_, layout) => {
    const field = (size = SIZE) => fieldOf(layout.compact, size)

    const at = (f: TileField, slot: Slot, tile: number) => (tile - 1) * attr(f, slot.name).itemSize + slot.offset

    const valueOf = (f: TileField, slot: Slot, tile: number) => attr(f, slot.name).array[at(f, slot, tile)]

    const regionOf = (f: TileField, tile: number) => {
        const start = at(f, layout.flag, tile)
        if (!layout.compact) return Array.from(attr(f, layout.flag.name).array.slice(start, start + 4))

        const material = f.displayPoints.material as THREE.ShaderMaterial
        const rectangles: ArrayLike<number> = (material.uniforms.flagRectangles.value as THREE.DataTexture).image.data
        const flag = attr(f, layout.flag.name).array[start]
        return Array.from(rectangles).slice(flag * 4, flag * 4 + 4)
    }

    describe("setOwners", () => {
        it("gives the tile the country's atlas region", () => {
            const f = field()
            const fr = regions.get("fr")!

            f.setOwners([{tile: 3, country: "fr"}])

            expect(regionOf(f, 3)).toEqual([fr.x, fr.y, fr.width, fr.height])
        })

        it("leaves every other tile untouched", () => {
            const f = field()
            f.setOwners([{tile: 3, country: "fr"}])

            for (const tile of [1, 2, 4]) expect(regionOf(f, tile)).toEqual(ZERO)
        })

        it("starts every tile unowned, which the shader draws blank", () => {
            const f = field()
            for (let tile = 1; tile <= SIZE; tile++) expect(regionOf(f, tile)).toEqual(ZERO)
        })

        it("leaves a tile blank and warns once for a country with no region", () => {
            const f = field()
            f.setOwners([{tile: 1, country: "zz"}, {tile: 2, country: "zz"}])

            expect(regionOf(f, 1)).toEqual(ZERO)
            expect(regionOf(f, 2)).toEqual(ZERO)
            expect(console.warn).toHaveBeenCalledTimes(1)
        })

        it("blanks the tile again when a rolled-back click leaves it unowned", () => {
            const f = field()
            f.setOwners([{tile: 3, country: "fr"}])

            f.setOwners([{tile: 3, country: undefined}])

            expect(regionOf(f, 3)).toEqual(ZERO)
            expect(console.warn).not.toHaveBeenCalled()
        })

        it("uploads the tile a rollback blanked", () => {
            const f = field()
            f.setOwners([{tile: 2, country: "fr"}])
            const flag = attr(f, layout.flag.name)
            flag.clearUpdateRanges()

            f.setOwners([{tile: 2, country: undefined}])

            expect(flag.updateRanges).toEqual([{start: at(f, layout.flag, 2), count: layout.flag.width}])
        })

        it("does not touch the buffer when nothing changed", () => {
            const f = field()
            const flag = attr(f, layout.flag.name)
            const version = flag.version

            f.setOwners([])
            expect(flag.updateRanges).toEqual([])
            expect(flag.version).toBe(version)
        })

        it("keeps the same attribute and backing array across changes", () => {
            const f = field()
            const flag = attr(f, layout.flag.name)
            const array = flag.array

            f.setOwners([{tile: 1, country: "fr"}])
            f.setOwners([{tile: 2, country: "jp"}])

            expect(attr(f, layout.flag.name)).toBe(flag)
            expect(attr(f, layout.flag.name).array).toBe(array)
        })

        it("uploads only the tiles that changed", () => {
            const f = field()
            const flag = attr(f, layout.flag.name)
            flag.clearUpdateRanges()
            const version = flag.version

            f.setOwners([{tile: 2, country: "fr"}])

            expect(flag.updateRanges).toEqual([{start: at(f, layout.flag, 2), count: layout.flag.width}])
            expect(flag.version).toBe(version + 1)
        })

        it("collapses a large batch into a single spanning range", () => {
            const f = field(1000)
            const flag = attr(f, layout.flag.name)
            flag.clearUpdateRanges()

            const changes = Array.from({length: 200}, (_, i) => ({tile: i + 1, country: "fr"}))
            f.setOwners(changes)

            expect(flag.updateRanges).toEqual([{start: 0, count: at(f, layout.flag, 200) + layout.flag.width}])
        })
    })

    describe("setLandmasses", () => {
        it("writes each tile's landmass where the shader reads it", () => {
            const f = field()
            const assignment = Uint16Array.from({length: SIZE}, (_, i) => i % 7)

            f.setLandmasses(assignment)

            for (let tile = 1; tile <= SIZE; tile++) expect(valueOf(f, layout.landmass, tile)).toBe((tile - 1) % 7)
        })

        it("keeps the flags, and uploads the whole buffer with their ranges pending", () => {
            const f = field()
            const fr = regions.get("fr")!
            f.setOwners([{tile: 2, country: "fr"}])

            f.setLandmasses(new Uint16Array(SIZE).fill(5))

            const landmass = attr(f, layout.landmass.name)
            expect(regionOf(f, 2)).toEqual([fr.x, fr.y, fr.width, fr.height])
            expect(landmass.updateRanges).toContainEqual({start: 0, count: landmass.array.length})
        })
    })

    describe("setHover", () => {
        it("marks the hovered tile", () => {
            const f = field()
            f.setHover(4)
            expect(valueOf(f, layout.hover, 4)).toBe(1)
        })

        it("clears the previously hovered tile", () => {
            const f = field()
            f.setHover(4)
            f.setHover(9)

            expect(valueOf(f, layout.hover, 4)).toBe(0)
            expect(valueOf(f, layout.hover, 9)).toBe(1)
        })

        it("clears everything when nothing is hovered", () => {
            const f = field()
            f.setHover(4)
            f.setHover(undefined)
            expect(Array.from(attr(f, layout.hover.name).array).every(v => v === 0)).toBe(true)
        })

        it("highlights exactly one tile at a time", () => {
            const f = field()
            for (const tile of [1, 7, 20, 32]) f.setHover(tile)

            const hovered = Array.from({length: SIZE}, (_, i) => i + 1).filter(tile => valueOf(f, layout.hover, tile) === 1)
            expect(hovered).toEqual([32])
        })

        it("does no work when the hovered tile repeats", () => {
            const f = field()
            f.setHover(4)
            const hover = attr(f, layout.hover.name)
            hover.clearUpdateRanges()

            expect(f.setHover(4)).toBe(false)
            expect(hover.updateRanges).toEqual([])
        })

        it("keeps the same attribute and backing array across moves", () => {
            const f = field()
            const hover = attr(f, layout.hover.name)
            const array = hover.array

            for (let tile = 1; tile <= SIZE; tile++) f.setHover(tile)

            expect(attr(f, layout.hover.name)).toBe(hover)
            expect(attr(f, layout.hover.name).array).toBe(array)
        })

        it("uploads only the two tiles involved", () => {
            const f = field()
            f.setHover(4)
            const hover = attr(f, layout.hover.name)
            hover.clearUpdateRanges()

            f.setHover(9)

            expect(hover.updateRanges).toEqual([
                {start: at(f, layout.hover, 4), count: 1},
                {start: at(f, layout.hover, 9), count: 1},
            ])
        })
    })

    describe("setShields", () => {
        it("writes each tile's shields where the shader reads them, and nothing else", () => {
            const f = field()

            f.setShields([{tile: 3, shields: 4, was: 0}, {tile: 5, shields: 10, was: 9}])
            f.setShields([{tile: 3, shields: 0, was: 4}])

            const values = Array.from(attr(f, layout.shield.name).array)
            expect(values[at(f, layout.shield, 5)]).toBe(10)
            values[at(f, layout.shield, 5)] = 0
            expect(values.every(v => v === 0)).toBe(true)
        })

        it("patches only the tiles that moved", () => {
            const f = field()
            const shield = attr(f, layout.shield.name)
            shield.clearUpdateRanges()

            f.setShields([{tile: 3, shields: 1, was: 0}, {tile: 9, shields: 2, was: 0}])

            expect(shield.updateRanges).toEqual([
                {start: at(f, layout.shield, 3), count: 1},
                {start: at(f, layout.shield, 9), count: 1},
            ])
        })

        it("patches one span for a load too big to list tile by tile", () => {
            const f = field()
            const shield = attr(f, layout.shield.name)
            shield.clearUpdateRanges()

            const many = Array.from({length: 100}, () => ({tile: 4, shields: 2, was: 1}))
            f.setShields([...many, {tile: 20, shields: 3, was: 0}])

            const start = at(f, layout.shield, 4)
            expect(shield.updateRanges).toEqual([{start, count: at(f, layout.shield, 20) + 1 - start}])
        })
    })

    describe("setPulses", () => {
        it("marks the tiles a flag still misses, and clears them again", () => {
            const f = field()

            f.setPulses([2, 6], true)
            expect([2, 6, 7].map(tile => valueOf(f, layout.pulse, tile))).toEqual([1, 1, 0])

            f.setPulses([2], false)
            expect([2, 6].map(tile => valueOf(f, layout.pulse, tile))).toEqual([0, 1])
        })

        it("patches only those tiles", () => {
            const f = field()
            const pulse = attr(f, layout.pulse.name)
            pulse.clearUpdateRanges()

            f.setPulses([2, 6], true)

            expect(pulse.updateRanges).toEqual([
                {start: at(f, layout.pulse, 2), count: 1},
                {start: at(f, layout.pulse, 6), count: 1},
            ])
        })
    })

    describe("geometry", () => {
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

        it("keeps every buffer's stride a multiple of 4 bytes, which Metal needs to read it as is", () => {
            const f = field()
            for (const points of [f.displayPoints, f.pickingPoints]) {
                for (const attribute of Object.values(points.geometry.attributes)) {
                    const {itemSize, array} = attribute as THREE.BufferAttribute
                    expect((itemSize * array.BYTES_PER_ELEMENT) % 4).toBe(0)
                }
            }
        })
    })
})

describe("the compact layout's cost", () => {
    const bytesPerTile = (compact: boolean) => {
        const f = fieldOf(compact)
        const arrays = new Set([f.displayPoints, f.pickingPoints].flatMap(points =>
            Object.values(points.geometry.attributes).map(attribute => (attribute as THREE.BufferAttribute).array)))
        return Array.from(arrays).reduce((sum, array) => sum + array.byteLength, 0) / SIZE
    }

    it("is 20 bytes a tile, against 56 for the plain one", () => {
        expect(bytesPerTile(false)).toBe(56)
        expect(bytesPerTile(true)).toBe(20)
    })

    it("names every flag in 16 bits", () => {
        expect(regions.size + 1).toBeLessThanOrEqual(2 ** 16)
    })
})

describe("picking", () => {
    it("gives the plain picking points one colour per tile, starting at id 1", () => {
        const f = fieldOf(false)
        const colors = f.pickingPoints.geometry.getAttribute("color").array as Float32Array

        expect(colors[0]).toBe(0)
        expect(colors[1]).toBe(0)
        expect(colors[2]).toBeCloseTo(1 / 255, 6)
        expect(colors.length).toBe(SIZE * 3)
    })

    it("uploads no colour in the compact layout: the shader makes it from the vertex number", () => {
        const f = fieldOf(true)
        expect(f.pickingPoints.geometry.getAttribute("color")).toBeUndefined()
        expect((f.pickingPoints.material as THREE.ShaderMaterial).defines.COMPACT).toBe(true)
    })
})
