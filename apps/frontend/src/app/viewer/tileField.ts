import * as THREE from "three";
import type {OwnerChange} from "../../domain/tileOwnership.ts";
import type {ShieldChange} from "../../domain/shields.ts";
import {regions} from "./atlas.ts";
import {warnOnce} from "../../domain/warnOnce.ts";
import {disposeMaterial} from "./scene.ts";
import {integerToColor} from "./pickingColors.ts";
import type {TileGeometry} from "./points.ts";
import type {Graphics} from "./graphics.ts";

import displayVertex from "./shaders/display/vertex.glsl"
import displayFragment from "./shaders/display/fragment.glsl"
import pickerVertex from "./shaders/picker/vertex.glsl"
import pickerFragment from "./shaders/picker/fragment.glsl"

const MAX_INDIVIDUAL_RANGES = 64

// Must match TILE_COLUMNS in shaders/territory/fragment.glsl.
export const TILE_COLUMNS = 512

const UNOWNED = 0

// The pulse slot holds four marks, one bit each: the display shaders read them back with floor and mod.
const FORTIFY_PULSE = 1
const FRONT_PULSE = 2
const MISSION_MARK = 4
const CAPITAL_MARK = 8

const RECTANGLE = 4

const FLAGS = new Map<string, number>()
const RECTANGLES = new Float32Array((regions.size + 1) * RECTANGLE)
Array.from(regions).forEach(([code, {x, y, width, height}], i) => {
    FLAGS.set(code, i + 1)
    RECTANGLES.set([x, y, width, height], (i + 1) * RECTANGLE)
})

type Slot = {attribute: THREE.BufferAttribute, offset: number, width: number}

type Slots = {flag: Slot, landmass: Slot, hover: Slot, shield: Slot, pulse: Slot}

type Layout = {slots: Slots, attributes: Record<string, THREE.BufferAttribute>}

export class TileField {
    readonly displayPoints: THREE.Points
    readonly pickingPoints: THREE.Points
    readonly size: number
    // Each tile's flag and landmass at its cell's texel, for the territory layer.
    readonly tileData: THREE.DataTexture
    readonly flagRectangles: THREE.DataTexture

    private readonly compact: boolean
    private readonly slots: Slots
    private readonly texels: Uint32Array
    private readonly tileValues: Uint16Array
    private readonly tileSource: THREE.DataTexture
    private changedRows: {from: number, to: number} | undefined
    private hovered: number | undefined

    constructor(
        uniforms: {[uniform: string]: THREE.IUniform},
        pickingUniforms: {[uniform: string]: THREE.IUniform},
        data: TileGeometry,
        graphics: Pick<Graphics, "compact">,
    ) {
        const {positions, size, cells} = data
        this.size = size
        this.compact = graphics.compact
        this.texels = cells.texels
        this.flagRectangles = flagRectangles()

        const rows = Math.ceil(size / TILE_COLUMNS)
        this.tileValues = new Uint16Array(TILE_COLUMNS * rows * 2)
        this.tileData = tileTexture(this.tileValues, rows)
        this.tileSource = tileTexture(this.tileValues, rows)

        const position = new THREE.BufferAttribute(positions, 3)
        const {slots, attributes} = this.compact ? compactLayout(size) : plainLayout(size)
        this.slots = slots

        const displayGeometry = new THREE.BufferGeometry()
        displayGeometry.setAttribute('position', position)
        for (const [name, attribute] of Object.entries(attributes)) displayGeometry.setAttribute(name, attribute)

        const pickingGeometry = new THREE.BufferGeometry()
        pickingGeometry.setAttribute('position', position)
        if (!this.compact) pickingGeometry.setAttribute('color', new THREE.BufferAttribute(pickingColors(size), 3))

        const defines = {COMPACT: this.compact}

        this.displayPoints = new THREE.Points(displayGeometry, new THREE.ShaderMaterial({
            transparent: true,
            defines,
            uniforms: this.compact ? {...uniforms, flagRectangles: {value: this.flagRectangles}} : uniforms,
            vertexShader: displayVertex,
            fragmentShader: displayFragment,
        }))

        this.pickingPoints = new THREE.Points(pickingGeometry, new THREE.ShaderMaterial({
            defines,
            uniforms: pickingUniforms,
            vertexShader: pickerVertex,
            fragmentShader: pickerFragment,
        }))
    }

    setOwners(changes: OwnerChange[]) {
        const tiles: number[] = []
        const flags: number[] = []
        for (const {tile, country} of changes) {
            const flag = country === undefined ? UNOWNED : FLAGS.get(country)
            if (flag === undefined) {
                warnOnce(`No sprite region for country "${country}", leaving its tiles blank`)
                continue
            }
            tiles.push(tile)
            flags.push(flag)
        }

        this.patch(this.slots.flag, tiles, (values, at, i) => {
            if (this.compact) values[at] = flags[i]
            else values.set(RECTANGLES.subarray(flags[i] * RECTANGLE, (flags[i] + 1) * RECTANGLE), at)
        })

        tiles.forEach((tile, i) => {
            const texel = this.texels[tile - 1]
            this.tileValues[texel * 2] = flags[i]
            const row = Math.floor(texel / TILE_COLUMNS)
            this.changedRows = {from: Math.min(row, this.changedRows?.from ?? row), to: Math.max(row, this.changedRows?.to ?? row)}
        })
    }

    setShields(changes: readonly ShieldChange[]) {
        this.patch(this.slots.shield, changes.map(({tile}) => tile), (values, at, i) => {
            values[at] = changes[i].shields
        })
    }

    setPulses(tiles: readonly number[], on: boolean) {
        this.mark(tiles, FORTIFY_PULSE, on)
    }

    setFront(tiles: readonly number[], on: boolean) {
        this.mark(tiles, FRONT_PULSE, on)
    }

    setMissionMarks(tiles: readonly number[], on: boolean) {
        this.mark(tiles, MISSION_MARK, on)
    }

    setCapitalMarks(tiles: readonly number[], on: boolean) {
        this.mark(tiles, CAPITAL_MARK, on)
    }

    setLandmasses(assignment: Uint16Array) {
        const {attribute, offset} = this.slots.landmass
        const values = attribute.array
        for (let i = 0; i < this.size; i++) values[i * attribute.itemSize + offset] = assignment[i]
        // The flags may share this buffer: with their ranges pending, a bare needsUpdate would upload only those.
        attribute.addUpdateRange(0, values.length)
        attribute.needsUpdate = true

        for (let i = 0; i < this.size; i++) this.tileValues[this.texels[i] * 2 + 1] = assignment[i]
        this.tileData.needsUpdate = true
    }

    // Sends the rows of tileData whose tiles changed hands, rather than the whole texture.
    uploadTiles(renderer: THREE.WebGLRenderer) {
        if (!this.changedRows) return
        const {from, to} = this.changedRows
        this.changedRows = undefined
        renderer.copyTextureToTexture(
            this.tileSource,
            this.tileData,
            new THREE.Box2(new THREE.Vector2(0, from), new THREE.Vector2(TILE_COLUMNS, to + 1)),
            new THREE.Vector2(0, from),
        )
    }

    setHover(tile: number | undefined): boolean {
        if (tile === this.hovered) return false

        const tiles = [this.hovered, tile].filter((index) => index !== undefined)
        this.patch(this.slots.hover, tiles, (values, at, i) => {
            values[at] = tiles[i] === tile ? 1 : 0
        })

        this.hovered = tile
        return true
    }

    dispose() {
        for (const points of [this.displayPoints, this.pickingPoints]) {
            points.geometry.dispose()
            disposeMaterial(points.material as THREE.Material)
        }
        this.tileData.dispose()
        this.flagRectangles.dispose()
    }

    private mark(tiles: readonly number[], bit: number, on: boolean) {
        this.patch(this.slots.pulse, tiles, (values, at) => {
            values[at] = on ? values[at] | bit : values[at] & ~bit
        })
    }

    private patch(slot: Slot, tiles: readonly number[], write: (values: THREE.TypedArray, at: number, i: number) => void) {
        if (tiles.length === 0) return

        const {attribute, offset, width} = slot
        const individual = tiles.length <= MAX_INDIVIDUAL_RANGES
        let lowest = Infinity
        let highest = -Infinity

        tiles.forEach((tile, i) => {
            const at = (tile - 1) * attribute.itemSize + offset
            write(attribute.array, at, i)

            if (individual) {
                attribute.addUpdateRange(at, width)
            } else {
                lowest = Math.min(lowest, at)
                highest = Math.max(highest, at + width)
            }
        })

        if (!individual) attribute.addUpdateRange(lowest, highest - lowest)
        attribute.needsUpdate = true
    }
}

function plainLayout(size: number): Layout {
    const floats = (width: number) => new THREE.BufferAttribute(new Float32Array(size * width), width)
    const [regionVector, landmassIndex, hover, shield, pulse] = [RECTANGLE, 1, 1, 1, 1].map(floats)
    const whole = (attribute: THREE.BufferAttribute) => ({attribute, offset: 0, width: attribute.itemSize})

    return {
        slots: {
            flag: whole(regionVector),
            landmass: whole(landmassIndex),
            hover: whole(hover),
            shield: whole(shield),
            pulse: whole(pulse),
        },
        attributes: {regionVector, landmassIndex, hover, shield, pulse},
    }
}

function compactLayout(size: number): Layout {
    const ids = new THREE.BufferAttribute(new Uint16Array(size * 2), 2)
    const marks = new THREE.BufferAttribute(new Uint8Array(size * 4), 4)
    const at = (attribute: THREE.BufferAttribute, offset: number) => ({attribute, offset, width: 1})

    return {
        slots: {
            flag: at(ids, 0),
            landmass: at(ids, 1),
            hover: at(marks, 0),
            pulse: at(marks, 1),
            shield: at(marks, 2),
        },
        attributes: {ids, marks},
    }
}

function flagRectangles(): THREE.DataTexture {
    const texture = new THREE.DataTexture(RECTANGLES, RECTANGLES.length / RECTANGLE, 1, THREE.RGBAFormat, THREE.FloatType)
    texture.needsUpdate = true
    return texture
}

function tileTexture(values: Uint16Array, rows: number): THREE.DataTexture {
    const texture = new THREE.DataTexture(values, TILE_COLUMNS, rows, THREE.RGIntegerFormat, THREE.UnsignedShortType)
    texture.internalFormat = "RG16UI"
    texture.needsUpdate = true
    return texture
}

function pickingColors(size: number): Float32Array {
    const colors = new Float32Array(size * 3)
    for (let i = 0; i < size; i++) {
        const [r, g, b] = integerToColor(i + 1)
        colors[i * 3] = r / 255
        colors[i * 3 + 1] = g / 255
        colors[i * 3 + 2] = b / 255
    }
    return colors
}
