import {beforeEach, describe, expect, it, vi} from "vitest"
import * as THREE from "three"
import {TILE_COLUMNS, TileField} from "./tileField.ts"
import {BLANK_REGION, FIRST_SOLD, FLAG_NUMBERS, flagNumberOf, regions, showPicture, SOLD_MOST} from "./atlas.ts"
import {MOST_HEIGHT, soldCellOf} from "./flagAtlas.ts"
import {resetWarnOnce} from "../../domain/warnOnce.ts"
import type {TileGeometry} from "./points.ts"
import {chunksOf} from "../../domain/tileChunks.ts"
import {threeLattice} from "../../domain/lattice.ts"

const SIZE = 32

type Slot = {name: string, offset: number, width: number}

type Layout = {compact: boolean, flag: Slot, province: Slot, hover: Slot, shield: Slot, pulse: Slot}

const PLAIN: Layout = {
    compact: false,
    flag: {name: "flagRegion", offset: 0, width: 2},
    province: {name: "tileMarks", offset: 0, width: 1},
    hover: {name: "tileMarks", offset: 1, width: 1},
    pulse: {name: "tileMarks", offset: 2, width: 1},
    shield: {name: "tileMarks", offset: 3, width: 1},
}

const COMPACT: Layout = {
    compact: true,
    flag: {name: "ids", offset: 0, width: 1},
    province: {name: "ids", offset: 1, width: 1},
    hover: {name: "marks", offset: 0, width: 1},
    pulse: {name: "marks", offset: 1, width: 1},
    shield: {name: "marks", offset: 2, width: 1},
}

function spiral(size: number): Float32Array {
    const positions = new Float32Array(size * 3)
    for (let i = 0; i < size; i++) {
        const y = 1 - 2 * (i + 0.5) / size
        const around = i * Math.PI * (3 - Math.sqrt(5))
        const r = Math.sqrt(1 - y * y)
        positions.set([Math.cos(around) * r, y, Math.sin(around) * r], i * 3)
    }
    return positions
}

function uniformsOf(flagPaint = 0) {
    return {
        pointSize: {value: 4},
        pixelsPerRadian: {value: 400},
        flagPaint: {value: flagPaint},
        blastRadii: {value: [0, 0, 0, 0]},
        motion: {value: 1},
    }
}

// The texels run backwards, so a test sees a tile written at its texel and not at its index.
function fieldOf(compact: boolean, size = SIZE, uniforms = uniformsOf()) {
    const positions = spiral(size)
    const texels = Uint32Array.from({length: size}, (_, i) => size - 1 - i)
    const data: TileGeometry = {
        positions,
        size,
        cells: {lattice: threeLattice(1), words: new Uint32Array(2), texels, tiles: Uint32Array.from({length: size}, (_, i) => size - 1 - i)},
        chunks: chunksOf(positions),
    }
    return new TileField(uniforms, data, {compact})
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
        if (!layout.compact) {
            const [a, b] = attr(f, layout.flag.name).array.slice(start, start + 2)
            return [a % 2048, Math.floor(a / 2048), b % 2048, Math.floor(b / 2048)]
        }

        const material = (f.displayPoints.material as THREE.ShaderMaterial[])[0]
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

        it("gives a flag with no picture the atlas's blank, over the flag it took the tile from", () => {
            const f = field()
            f.setOwners([{tile: 1, country: "fr"}])

            f.setOwners([{tile: 1, country: "r-k3p9x2ma0b"}, {tile: 2, country: "zz"}])

            const {x, y, width, height} = BLANK_REGION
            expect(regionOf(f, 1)).toEqual([x, y, width, height])
            expect(regionOf(f, 2)).toEqual([x, y, width, height])
            expect(console.warn).not.toHaveBeenCalled()
        })

        it("gives a sold flag's tiles its cell once its picture is shown, and leaves the others", () => {
            const f = field()
            f.setOwners([{tile: 1, country: "r-cell000000"}, {tile: 2, country: "r-cell000000"}, {tile: 3, country: "fr"}])
            const number = flagNumberOf("r-cell000000")
            const cell = {x: 300, y: 1348, width: 100, height: 67}

            showPicture(number, cell)
            f.showFlag(number)
            showPicture(number, undefined)

            for (const tile of [1, 2]) expect(regionOf(f, tile)).toEqual([cell.x, cell.y, cell.width, cell.height])
            const fr = regions.get("fr")!
            expect(regionOf(f, 3)).toEqual([fr.x, fr.y, fr.width, fr.height])
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

    describe("setProvinces", () => {
        it("writes each tile's province where the shader reads it", () => {
            const f = field()
            const assignment = Uint16Array.from({length: SIZE}, (_, i) => i % 7)

            f.setProvinces(assignment)

            for (let tile = 1; tile <= SIZE; tile++) expect(valueOf(f, layout.province, tile)).toBe((tile - 1) % 7)
        })

        it("keeps the flags, and uploads the whole buffer with their ranges pending", () => {
            const f = field()
            const fr = regions.get("fr")!
            f.setOwners([{tile: 2, country: "fr"}])

            f.setProvinces(new Uint16Array(SIZE).fill(5))

            const province = attr(f, layout.province.name)
            expect(regionOf(f, 2)).toEqual([fr.x, fr.y, fr.width, fr.height])
            expect(province.updateRanges).toContainEqual({start: 0, count: province.array.length})
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

        it("patches only the tiles that moved, however long the list", () => {
            const f = field()
            const shield = attr(f, layout.shield.name)
            shield.clearUpdateRanges()

            const many = Array.from({length: 100}, () => ({tile: 4, shields: 2, was: 1}))
            f.setShields([...many, {tile: 20, shields: 3, was: 0}])

            expect(shield.updateRanges).toEqual([
                {start: at(f, layout.shield, 4), count: 1},
                {start: at(f, layout.shield, 20), count: 1},
            ])
        })

        it("joins the nearest tiles of a scattered load into a few ranges", () => {
            const f = field(1000)
            const shield = attr(f, layout.shield.name)
            shield.clearUpdateRanges()

            f.setShields(Array.from({length: 300}, (_, i) => ({tile: 1 + i * 3, shields: 1, was: 0})))

            expect(shield.updateRanges.length).toBeLessThanOrEqual(64)
            expect(shield.updateRanges[0].start).toBe(at(f, layout.shield, 1))
            const last = shield.updateRanges[shield.updateRanges.length - 1]
            expect(last.start + last.count).toBe(at(f, layout.shield, 898) + 1)
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

        it("keeps each mark on a bit of its own, so a capital and the front share a tile", () => {
            const f = field()

            f.setFront([3], true)
            f.setMissionMarks([3], true)
            f.setCapitalMarks([3, 5], true)
            expect([3, 5].map(tile => valueOf(f, layout.pulse, tile))).toEqual([2 + 4 + 8, 8])

            f.setCapitalMarks([3], false)
            f.setFront([3], false)
            expect(valueOf(f, layout.pulse, 3)).toBe(4)
        })

        it("takes every mark off some tiles, and puts them back as they were", () => {
            const f = field()
            f.setFront([3], true)
            f.setCapitalMarks([3, 5], true)
            f.setMineMarks([6], true)

            const worn = f.takeMarks([3, 4, 5, 6])
            expect(worn).toEqual([2 + 8, 0, 8, 32])
            expect([3, 4, 5, 6].map(tile => valueOf(f, layout.pulse, tile))).toEqual([0, 0, 0, 0])
            expect(valueOf(f, layout.pulse, 2), "a tile it was not given").toBe(0)

            f.setCapitalMarks([4], true)
            f.putMarks([3, 4, 5, 6], worn)
            expect([3, 4, 5, 6].map(tile => valueOf(f, layout.pulse, tile))).toEqual([2 + 8, 0, 8, 32])
        })
    })

    describe("geometry", () => {
        it("keeps every buffer's stride a multiple of 4 bytes, which Metal needs to read it as is", () => {
            const f = field()
            for (const attribute of Object.values(f.displayPoints.geometry.attributes)) {
                const {itemSize, array} = attribute as THREE.BufferAttribute
                expect((itemSize * array.BYTES_PER_ELEMENT) % 4).toBe(0)
            }
        })
    })
})

describe("what a tile costs", () => {
    const bytesPerTile = (compact: boolean) => {
        const f = fieldOf(compact)
        const arrays = Object.values(f.displayPoints.geometry.attributes).map(attribute => (attribute as THREE.BufferAttribute).array)
        return arrays.reduce((sum, array) => sum + array.byteLength, 0) / SIZE
    }

    it("is 20 bytes in the compact layout, and 36 in the plain one, of 32-bit floats alone", () => {
        expect(bytesPerTile(false)).toBe(36)
        expect(bytesPerTile(true)).toBe(20)
        for (const attribute of Object.values(fieldOf(false).displayPoints.geometry.attributes)) {
            expect((attribute as THREE.BufferAttribute).array).toBeInstanceOf(Float32Array)
        }
    })

    it("packs every atlas rectangle into floats exactly, the sold flags' cells under it too", () => {
        for (const {x, y, width, height} of regions.values()) {
            expect(Math.max(x, y, width, height)).toBeLessThan(2048)
            expect(Math.fround(x + y * 2048)).toBe(x + y * 2048)
        }
        const last = soldCellOf(FIRST_SOLD + SOLD_MOST - 1, {width: 1, height: 1})
        expect(last.y + last.height).toBeLessThan(MOST_HEIGHT)
        expect(Math.fround(last.x + last.y * 2048)).toBe(last.x + last.y * 2048)
    })

    it("names every flag in 16 bits", () => {
        expect(FLAG_NUMBERS - 1).toBeLessThanOrEqual(2 ** 16 - 1)
    })
})

describe("the territory layer's tile data", () => {
    const SIZE_OVER_ROWS = TILE_COLUMNS * 2 + 10
    const flagNumberOf = (code: string) => Array.from(regions.keys()).indexOf(code) + 1
    const valuesOf = (f: TileField) => f.tileData.image.data as unknown as Uint16Array
    const texelOf = (tile: number) => SIZE_OVER_ROWS - tile

    it.each([false, true])("holds each tile's flag at its cell's texel, compact %s", (compact) => {
        const f = fieldOf(compact, SIZE_OVER_ROWS)

        f.setOwners([{tile: 3, country: "fr"}, {tile: 4, country: "jp"}])
        f.setOwners([{tile: 4, country: undefined}])

        expect(valuesOf(f)[texelOf(3) * 2]).toBe(flagNumberOf("fr"))
        expect(valuesOf(f)[texelOf(4) * 2]).toBe(0)
        expect(valuesOf(f)[2 * 2]).toBe(0)
    })

    it("holds each tile's province beside its flag", () => {
        const f = fieldOf(true, SIZE_OVER_ROWS)

        f.setProvinces(Uint16Array.from({length: SIZE_OVER_ROWS}, (_, i) => i % 7))

        for (const tile of [1, 2, 600, SIZE_OVER_ROWS]) expect(valuesOf(f)[texelOf(tile) * 2 + 1]).toBe((tile - 1) % 7)
    })

    it("sends only the rows whose tiles changed hands, once", () => {
        const f = fieldOf(true, SIZE_OVER_ROWS)
        const copyTextureToTexture = vi.fn()
        const renderer = {copyTextureToTexture} as unknown as THREE.WebGLRenderer

        f.setOwners([{tile: 1, country: "fr"}, {tile: 5, country: "fr"}])
        f.uploadTiles(renderer)
        f.uploadTiles(renderer)

        expect(copyTextureToTexture).toHaveBeenCalledTimes(1)
        const [, target, region, position] = copyTextureToTexture.mock.calls[0]
        expect(target).toBe(f.tileData)
        expect([region.min.y, region.max.y, region.min.x, region.max.x]).toEqual([2, 3, 0, TILE_COLUMNS])
        expect(position.y).toBe(2)
    })

    it("sends two rows far apart, and not the rows between them", () => {
        const f = fieldOf(true, SIZE_OVER_ROWS)
        const copyTextureToTexture = vi.fn()
        const renderer = {copyTextureToTexture} as unknown as THREE.WebGLRenderer

        f.setOwners([{tile: 1, country: "fr"}, {tile: SIZE_OVER_ROWS, country: "fr"}])
        f.uploadTiles(renderer)

        const rows = copyTextureToTexture.mock.calls.map(([, , region]) => [region.min.y, region.max.y])
        expect(rows).toEqual([[0, 1], [2, 3]])
    })
})

describe("the tiles drawn", () => {
    const SIZE_DRAWN = 2000
    const positions = spiral(SIZE_DRAWN)
    const groupsOf = (f: TileField) => f.displayPoints.geometry.groups.map(({start, count}) => [start, start + count])
    const drawn = (f: TileField, tile: number) => groupsOf(f).some(([from, to]) => from <= tile - 1 && tile - 1 < to)

    const cameraAt = (tile: number, zoom: number) => {
        const camera = new THREE.OrthographicCamera(-1.6, 1.6, 1, -1, 0.01, 100)
        camera.position.set(positions[tile * 3 - 3], positions[tile * 3 - 2], positions[tile * 3 - 1]).multiplyScalar(5)
        camera.lookAt(0, 0, 0)
        camera.zoom = zoom
        camera.updateProjectionMatrix()
        return camera
    }

    it("are the pieces in sight, not those behind the earth or off screen", () => {
        const f = fieldOf(true, SIZE_DRAWN)
        f.cull(cameraAt(500, 10))

        expect(drawn(f, 500)).toBe(true)
        expect(drawn(f, SIZE_DRAWN - 500)).toBe(false)
        const tiles = groupsOf(f).reduce((sum, [from, to]) => sum + to - from, 0)
        expect(tiles).toBeLessThan(SIZE_DRAWN / 4)
    })

    it("are none while the painted flags cover the globe and no tile wears a ring", () => {
        const f = fieldOf(true, SIZE_DRAWN, uniformsOf(1))
        f.setMissionMarks([500], true)
        f.setCapitalMarks([500], true)
        f.cull(cameraAt(500, 1))

        expect(f.displayPoints.geometry.groups).toEqual([])
    })

    it("are the pieces with a ring under the painted flags, until the ring goes", () => {
        const f = fieldOf(true, SIZE_DRAWN, uniformsOf(1))
        const camera = cameraAt(500, 1)

        f.setFront([500, 501], true)
        f.setPulses([500], true)
        f.cull(camera)
        expect(drawn(f, 500)).toBe(true)
        expect(drawn(f, 10)).toBe(false)

        f.setFront([500, 501], false)
        f.cull(camera)
        expect(drawn(f, 500)).toBe(true)

        f.setPulses([500], false)
        f.cull(camera)
        expect(f.displayPoints.geometry.groups).toEqual([])
    })

    it("stay the pieces found while the camera moves less than the slack, and hold every tile in sight", () => {
        const f = fieldOf(true, SIZE_DRAWN)
        const camera = cameraAt(500, 10)
        f.cull(camera)
        const found = f.displayPoints.geometry.groups.map(({start, count}) => ({start, count}))

        const turn = new THREE.Vector3(0, 1, 0)
        for (let step = 0; step < 20; step++) {
            camera.position.applyAxisAngle(turn, 0.001)
            camera.lookAt(0, 0, 0)
            f.cull(camera)
            camera.updateMatrixWorld()
            const toClip = new THREE.Matrix4().multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)
            const back = new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 2)
            for (let tile = 1; tile <= SIZE_DRAWN; tile++) {
                const ground = new THREE.Vector3(positions[tile * 3 - 3], positions[tile * 3 - 2], positions[tile * 3 - 1])
                const clip = ground.clone().applyMatrix4(toClip)
                const inSight = ground.dot(back) > 0 && Math.abs(clip.x) <= 1 && Math.abs(clip.y) <= 1
                if (inSight) expect(drawn(f, tile), `tile ${tile} after ${step + 1} steps`).toBe(true)
            }
        }
        expect(f.displayPoints.geometry.groups.map(({start, count}) => ({start, count}))).toEqual(found)
    })

    it("are found again once the camera has moved past the slack", () => {
        const f = fieldOf(true, SIZE_DRAWN)
        f.cull(cameraAt(500, 10))
        f.cull(cameraAt(1500, 10))
        expect(drawn(f, 1500)).toBe(true)
        expect(drawn(f, 500)).toBe(false)
    })

    it("are every tile, in one draw, when the screen shows most of the globe", () => {
        const f = fieldOf(true, SIZE_DRAWN, uniformsOf(0.5))
        f.cull(cameraAt(500, 1))
        expect(groupsOf(f)).toEqual([[0, SIZE_DRAWN]])
    })
})
