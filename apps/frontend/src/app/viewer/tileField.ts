import * as THREE from "three";
import type {OwnerChange} from "../../domain/tileOwnership.ts";
import type {ShieldChange} from "../../domain/shields.ts";
import {regions} from "./atlas.ts";
import {warnOnce} from "../../domain/warnOnce.ts";
import {disposeMaterial} from "./scene.ts";
import {integerToColor} from "./pickingColors.ts";
import type {PointGeometryData} from "./coordinatesBinary.ts";

import displayVertex from "./shaders/display/vertex.glsl"
import displayFragment from "./shaders/display/fragment.glsl"
import pickerVertex from "./shaders/picker/vertex.glsl"
import pickerFragment from "./shaders/picker/fragment.glsl"

const REGION_STRIDE = 4

const MAX_INDIVIDUAL_RANGES = 64

const UNOWNED = {x: 0, y: 0, width: 0, height: 0}

export class TileField {
    readonly displayPoints: THREE.Points
    readonly pickingPoints: THREE.Points
    readonly size: number

    private readonly regionVector: THREE.BufferAttribute
    private readonly landmass: THREE.BufferAttribute
    private readonly hover: THREE.BufferAttribute
    private readonly shield: THREE.BufferAttribute
    private readonly pulse: THREE.BufferAttribute
    private hovered: number | undefined

    constructor(
        uniforms: {[uniform: string]: THREE.IUniform},
        pickingUniforms: {[uniform: string]: THREE.IUniform},
        data: PointGeometryData,
        lit: boolean,
    ) {
        const {positions, size} = data
        this.size = size

        const position = new THREE.BufferAttribute(positions, 3)

        this.regionVector = new THREE.BufferAttribute(new Float32Array(size * REGION_STRIDE), REGION_STRIDE)
        this.landmass = new THREE.BufferAttribute(new Float32Array(size), 1)
        this.hover = new THREE.BufferAttribute(new Float32Array(size), 1)
        this.shield = new THREE.BufferAttribute(new Float32Array(size), 1)
        this.pulse = new THREE.BufferAttribute(new Float32Array(size), 1)

        const displayGeometry = new THREE.BufferGeometry()
        displayGeometry.setAttribute('position', position)
        displayGeometry.setAttribute('regionVector', this.regionVector)
        displayGeometry.setAttribute('landmassIndex', this.landmass)
        displayGeometry.setAttribute('hover', this.hover)
        displayGeometry.setAttribute('shield', this.shield)
        displayGeometry.setAttribute('pulse', this.pulse)

        const pickingGeometry = new THREE.BufferGeometry()
        pickingGeometry.setAttribute('position', position)
        pickingGeometry.setAttribute('color', new THREE.BufferAttribute(pickingColors(size), 3))

        this.displayPoints = new THREE.Points(displayGeometry, new THREE.ShaderMaterial({
            transparent: true,
            defines: {LIT: lit},
            uniforms,
            vertexShader: displayVertex,
            fragmentShader: displayFragment,
        }))

        this.pickingPoints = new THREE.Points(pickingGeometry, new THREE.ShaderMaterial({
            uniforms: pickingUniforms,
            vertexShader: pickerVertex,
            fragmentShader: pickerFragment,
        }))
    }

    setOwners(changes: OwnerChange[]) {
        if (changes.length === 0) return

        const values = this.regionVector.array as Float32Array
        const individual = changes.length <= MAX_INDIVIDUAL_RANGES
        let lowest = Infinity
        let highest = -Infinity

        for (const {tile, country} of changes) {
            const region = country === undefined ? UNOWNED : regions.get(country)
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

    setShields(changes: readonly ShieldChange[]) {
        if (changes.length === 0) return

        const values = this.shield.array as Float32Array
        const individual = changes.length <= MAX_INDIVIDUAL_RANGES
        let lowest = Infinity
        let highest = -Infinity

        for (const {tile, shields} of changes) {
            const index = tile - 1
            values[index] = shields

            if (individual) {
                this.shield.addUpdateRange(index, 1)
            } else {
                lowest = Math.min(lowest, index)
                highest = Math.max(highest, index + 1)
            }
        }

        if (!individual && highest > lowest) {
            this.shield.addUpdateRange(lowest, highest - lowest)
        }
        this.shield.needsUpdate = true
    }

    setPulses(tiles: readonly number[], on: boolean) {
        if (tiles.length === 0) return

        const values = this.pulse.array as Float32Array
        const individual = tiles.length <= MAX_INDIVIDUAL_RANGES
        let lowest = Infinity
        let highest = -Infinity

        for (const tile of tiles) {
            const index = tile - 1
            values[index] = on ? 1 : 0

            if (individual) {
                this.pulse.addUpdateRange(index, 1)
            } else {
                lowest = Math.min(lowest, index)
                highest = Math.max(highest, index + 1)
            }
        }

        if (!individual && highest > lowest) {
            this.pulse.addUpdateRange(lowest, highest - lowest)
        }
        this.pulse.needsUpdate = true
    }

    setLandmasses(assignment: Uint16Array) {
        const values = this.landmass.array as Float32Array
        for (let i = 0; i < values.length; i++) values[i] = assignment[i]
        this.landmass.needsUpdate = true
    }

    setHover(tile: number | undefined): boolean {
        if (tile === this.hovered) return false

        const values = this.hover.array as Float32Array
        for (const index of [this.hovered, tile]) {
            if (index === undefined) continue
            values[index - 1] = index === tile ? 1 : 0
            this.hover.addUpdateRange(index - 1, 1)
        }

        this.hover.needsUpdate = true
        this.hovered = tile
        return true
    }

    dispose() {
        for (const points of [this.displayPoints, this.pickingPoints]) {
            points.geometry.dispose()
            disposeMaterial(points.material as THREE.Material)
        }
    }
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
