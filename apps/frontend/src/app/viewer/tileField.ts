import * as THREE from "three";
import type {OwnerChange} from "../../domain/tileOwnership.ts";
import {regions} from "./atlas.ts";
import {warnOnce} from "../../domain/warnOnce.ts";
import {disposeMaterial} from "./scene.ts";
import {integerToColor} from "./pickingColors.ts";
import type {PointGeometryData} from "./coordinatesBinary.ts";

import displayVertex from "./shaders/display/vertex.glsl"
import displayFragment from "./shaders/display/fragment.glsl"
import pickerVertex from "./shaders/picker/vertex.glsl"
import pickerFragment from "./shaders/picker/fragment.glsl"

/** Values per tile in the regionVector attribute: x, y, width, height. */
const REGION_STRIDE = 4

/**
 * Above this many changed tiles, one range covering the whole span beats
 * handing the renderer a range per tile for it to sort and merge. Ownership
 * batches are contiguous runs of tile ids, so their span is the batch itself;
 * live updates are a handful of scattered tiles, well under the threshold.
 */
const MAX_INDIVIDUAL_RANGES = 64

/**
 * The points that make up the globe, and the two attributes that change while
 * it is running: which country owns each tile, and which tile is hovered.
 *
 * Both attributes used to be rewritten wholesale on every change. Hovering
 * allocated a fresh 258k-element Float32Array, filled it with zeros, set one
 * element, and installed it as a *new* BufferAttribute — so moving the mouse
 * rebuilt and re-uploaded a megabyte per event, at whatever rate the mouse
 * reports. Ownership re-uploaded its 4 MB buffer on every 100 ms batch. Both
 * now mutate in place and tell the renderer only the ranges that moved.
 */
export class TileField {
    readonly displayPoints: THREE.Points
    readonly pickingPoints: THREE.Points
    readonly size: number

    private readonly regionVector: THREE.BufferAttribute
    private readonly hover: THREE.BufferAttribute
    private hovered: number | undefined

    constructor(uniforms: {[uniform: string]: THREE.IUniform}, data: PointGeometryData) {
        const {positions, size} = data
        this.size = size

        const position = new THREE.BufferAttribute(positions, 3)

        /** Zero width/height means "unowned", which the fragment shader draws blank. */
        this.regionVector = new THREE.BufferAttribute(new Float32Array(size * REGION_STRIDE), REGION_STRIDE)
        this.hover = new THREE.BufferAttribute(new Float32Array(size), 1)

        const displayGeometry = new THREE.BufferGeometry()
        displayGeometry.setAttribute('position', position)
        displayGeometry.setAttribute('regionVector', this.regionVector)
        displayGeometry.setAttribute('hover', this.hover)

        const pickingGeometry = new THREE.BufferGeometry()
        pickingGeometry.setAttribute('position', position)
        pickingGeometry.setAttribute('color', new THREE.BufferAttribute(pickingColors(size), 3))

        this.displayPoints = new THREE.Points(displayGeometry, new THREE.ShaderMaterial({
            transparent: true,
            uniforms,
            vertexShader: displayVertex,
            fragmentShader: displayFragment,
        }))

        this.pickingPoints = new THREE.Points(pickingGeometry, new THREE.ShaderMaterial({
            uniforms,
            vertexShader: pickerVertex,
            fragmentShader: pickerFragment,
        }))
    }

    /** Paints the tiles whose owner changed. */
    setOwners(changes: OwnerChange[]) {
        if (changes.length === 0) return

        const values = this.regionVector.array as Float32Array
        const individual = changes.length <= MAX_INDIVIDUAL_RANGES
        let lowest = Infinity
        let highest = -Infinity

        for (const {tile, country} of changes) {
            const region = regions.get(country)
            if (!region) {
                warnOnce(`No sprite region for country "${country}", leaving its tiles blank`)
                continue
            }

            const offset = (tile - 1) * REGION_STRIDE
            values[offset] = region.x
            values[offset + 1] = region.y
            values[offset + 2] = region.width
            values[offset + 3] = region.height

            if (individual) {
                this.regionVector.addUpdateRange(offset, REGION_STRIDE)
            } else {
                lowest = Math.min(lowest, offset)
                highest = Math.max(highest, offset + REGION_STRIDE)
            }
        }

        if (!individual && highest > lowest) {
            this.regionVector.addUpdateRange(lowest, highest - lowest)
        }
        this.regionVector.needsUpdate = true
    }

    /** Highlights one tile, or none. Repeating the current tile costs nothing. */
    setHover(tile: number | undefined) {
        if (tile === this.hovered) return

        const values = this.hover.array as Float32Array
        for (const index of [this.hovered, tile]) {
            if (index === undefined) continue
            values[index - 1] = index === tile ? 1 : 0
            this.hover.addUpdateRange(index - 1, 1)
        }

        this.hover.needsUpdate = true
        this.hovered = tile
    }

    dispose() {
        for (const points of [this.displayPoints, this.pickingPoints]) {
            points.geometry.dispose()
            disposeMaterial(points.material as THREE.Material)
        }
    }
}

/**
 * One colour per tile, so a picking render can be read back as a tile id.
 * Ids start at 1: black is the background and the inner sphere.
 */
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
