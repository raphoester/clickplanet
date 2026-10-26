import * as THREE from "three";
import type {OwnerChange} from "../../domain/tileOwnership.ts";
import type {ShieldChange} from "../../domain/shields.ts";
import {regions} from "./atlas.ts";
import {warnOnce} from "../../domain/warnOnce.ts";
import {disposeMaterial} from "./scene.ts";
import {integerToColor} from "./pickingColors.ts";
import type {PointGeometryData} from "./coordinatesBinary.ts";
import type {Graphics} from "./graphics.ts";

import displayVertex from "./shaders/display/vertex.glsl"
import displayFragment from "./shaders/display/fragment.glsl"
import pickerVertex from "./shaders/picker/vertex.glsl"
import pickerFragment from "./shaders/picker/fragment.glsl"

const MAX_INDIVIDUAL_RANGES = 64

const UNOWNED = 0

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

    private readonly compact: boolean
    private readonly slots: Slots
    private hovered: number | undefined

    constructor(
        uniforms: {[uniform: string]: THREE.IUniform},
        pickingUniforms: {[uniform: string]: THREE.IUniform},
        data: PointGeometryData,
        graphics: Pick<Graphics, "tiles" | "compact">,
    ) {
        const {positions, size} = data
        this.size = size
        this.compact = graphics.compact

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
            defines: {...defines, LIT: graphics.tiles},
            uniforms: this.compact ? {...uniforms, flagRectangles: {value: flagRectangles()}} : uniforms,
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
    }

    setShields(changes: readonly ShieldChange[]) {
        this.patch(this.slots.shield, changes.map(({tile}) => tile), (values, at, i) => {
            values[at] = changes[i].shields
        })
    }

    setPulses(tiles: readonly number[], on: boolean) {
        this.patch(this.slots.pulse, tiles, (values, at) => {
            values[at] = on ? 1 : 0
        })
    }

    setLandmasses(assignment: Uint16Array) {
        const {attribute, offset} = this.slots.landmass
        const values = attribute.array
        for (let i = 0; i < this.size; i++) values[i * attribute.itemSize + offset] = assignment[i]
        // The flags may share this buffer: with their ranges pending, a bare needsUpdate would upload only those.
        attribute.addUpdateRange(0, values.length)
        attribute.needsUpdate = true
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
