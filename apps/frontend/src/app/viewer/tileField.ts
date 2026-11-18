import * as THREE from "three";
import type {OwnerChange} from "../../domain/tileOwnership.ts";
import type {ShieldChange} from "../../domain/shields.ts";
import {flagNumbers, regions} from "./atlas.ts";
import {warnOnce} from "../../domain/warnOnce.ts";
import {disposeMaterial} from "./scene.ts";
import type {TileGeometry} from "./points.ts";
import type {Graphics} from "./graphics.ts";
import {chunkCount, chunkOf, type ChunkView, drawnRanges, showingChunks, type TileChunks} from "../../domain/tileChunks.ts";
import {joinSpans, type Span, spansOf} from "../../domain/spans.ts";

import displayVertex from "./shaders/display/vertex.glsl"
import displayFragment from "./shaders/display/fragment.glsl"

const MAX_UPDATE_RANGES = 64
// Tiles this close in a buffer go up in one range with the values between them.
const JOINED_TILES = 2

// A tile drawn for nothing costs the GPU less than one more draw call costs the CPU.
const MAX_DRAWS = 16
const LEAST_GAP = 8192

// Past this share of the globe's disc on screen, the cull skips little more than the shader's own far-side drop.
const CULL_BELOW = 0.5

// In screen half-heights: the pieces are found with this much to spare, and kept until the view has moved by it.
const CULL_SLACK = 0.25

const MAX_ROW_UPLOADS = 16

// Must match TILE_COLUMNS in shaders/territory/fragment.glsl.
// 1,024 columns: earth-2's 1,004,682 tiles fill 982 rows; at 512 they would fill 1,963 of the 2,048 WebGL2 guarantees.
export const TILE_COLUMNS = 1024

// Must match LIMB in shaders/display/vertex.glsl, and the throw it adds.
const LIMB = 0.05
const THROW = 0.6

// A sprite's width in tile sizes at most: a blast swells it up to 4 times, a virus's pods 1.8 times.
const SPRITE_MOST = 7.2

const UNOWNED = 0

// The pulse slot holds six marks, one bit each: the display shaders read them back with floor and mod.
const FORTIFY_PULSE = 1
const FRONT_PULSE = 2
const MISSION_MARK = 4
const CAPITAL_MARK = 8
// Must match marked(16.0) in shaders/display.
const INFECTED_MARK = 16
// Must match marked(32.0) in shaders/display.
const MINE_MARK = 32

// The only marks a tile draws while the painted flags cover it: everything else it draws fades with it.
const RINGS = FORTIFY_PULSE | FRONT_PULSE

const RECTANGLE = 4

// Must match PACK in shaders/display/vertex.glsl: the plain layout holds a rectangle as x + y * PACK, width + height * PACK.
const PACK = 2048

const RECTANGLES = new Float32Array((regions.size + 1) * RECTANGLE)
const PACKED = new Float32Array((regions.size + 1) * 2)
Array.from(regions.values()).forEach(({x, y, width, height}, i) => {
    RECTANGLES.set([x, y, width, height], (i + 1) * RECTANGLE)
    PACKED.set([x + y * PACK, width + height * PACK], (i + 1) * 2)
})

type Slot = {attribute: THREE.BufferAttribute, offset: number, width: number}

type Slots = {flag: Slot, landmass: Slot, hover: Slot, shield: Slot, pulse: Slot}

type Layout = {slots: Slots, attributes: Record<string, THREE.BufferAttribute>}

// The pieces in sight, and the view they were found for.
type Sighting = {
    ranges: Span[]
    right: THREE.Vector3
    up: THREE.Vector3
    back: THREE.Vector3
    sx: number, sy: number, tx: number, ty: number
    limb: number, reach: number, slack: number
}

export type TileUniforms = {
    pointSize: THREE.IUniform<number>
    pixelsPerRadian: THREE.IUniform<number>
    flagPaint: THREE.IUniform<number>
    blastRadii: THREE.IUniform<number[]>
    motion: THREE.IUniform<number>
}

export class TileField {
    readonly displayPoints: THREE.Points
    readonly size: number
    // Each tile's flag and landmass at its cell's texel, for the territory layer.
    readonly tileData: THREE.DataTexture
    readonly flagRectangles: THREE.DataTexture

    private readonly uniforms: TileUniforms
    private readonly compact: boolean
    private readonly slots: Slots
    private readonly texels: Uint32Array
    private readonly tileValues: Uint16Array
    private readonly tileSource: THREE.DataTexture
    private readonly changedRows = new Set<number>()
    private readonly chunks: TileChunks
    private readonly shown: Uint32Array
    // How many tiles of each chunk wear a ring, and the ranges of those chunks, until a ring changes.
    private readonly ringed: Uint32Array
    private rings: Span[] | undefined
    private sighting: Sighting | undefined
    private readonly toClip = new THREE.Matrix4()
    private hovered: number | undefined

    constructor(
        uniforms: TileUniforms & {[uniform: string]: THREE.IUniform},
        data: TileGeometry,
        graphics: Pick<Graphics, "compact">,
    ) {
        const {positions, size, cells, chunks} = data
        this.size = size
        this.uniforms = uniforms
        this.compact = graphics.compact
        this.texels = cells.texels
        this.chunks = chunks
        this.shown = new Uint32Array(chunkCount(chunks))
        this.ringed = new Uint32Array(chunkCount(chunks))
        this.flagRectangles = flagRectangles()

        const rows = Math.ceil(size / TILE_COLUMNS)
        this.tileValues = new Uint16Array(TILE_COLUMNS * rows * 2)
        this.tileData = tileTexture(this.tileValues, rows)
        this.tileSource = tileTexture(this.tileValues, rows)

        const {slots, attributes} = this.compact ? compactLayout(size) : plainLayout(size)
        this.slots = slots

        const geometry = new THREE.BufferGeometry()
        geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3))
        for (const [name, attribute] of Object.entries(attributes)) geometry.setAttribute(name, attribute)

        // An array of one material, so the geometry's groups are what is drawn: the pieces in sight.
        this.displayPoints = new THREE.Points(geometry, [new THREE.ShaderMaterial({
            transparent: true,
            defines: {COMPACT: this.compact},
            uniforms: this.compact ? {...uniforms, flagRectangles: {value: this.flagRectangles}} : uniforms,
            vertexShader: displayVertex,
            fragmentShader: displayFragment,
        })])
    }

    cull(camera: THREE.OrthographicCamera) {
        const geometry = this.displayPoints.geometry
        geometry.clearGroups()
        if (this.uniforms.flagPaint.value >= 1) {
            for (const {start, count} of this.ringRanges()) geometry.addGroup(start, count, 0)
            return
        }

        camera.updateMatrixWorld()
        this.toClip.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)
        const m = this.toClip.elements
        const width = 2 / Math.hypot(m[0], m[4], m[8]), height = 2 / Math.hypot(m[1], m[5], m[9])
        if (width * height >= CULL_BELOW * Math.PI) {
            geometry.addGroup(0, this.size, 0)
            return
        }

        const pointSize = this.uniforms.pointSize.value
        const perUnit = Math.max(this.uniforms.pixelsPerRadian.value, 1)
        const thrown = this.uniforms.motion.value * Math.max(0, ...this.uniforms.blastRadii.value) * THROW
        const sx = 2 / width, sy = 2 / height
        const now = {
            right: new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 0).normalize(),
            up: new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 1).normalize(),
            back: new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 2).normalize(),
            sx, sy, tx: m[12] / sx, ty: m[13] / sy,
            limb: LIMB + (pointSize * 0.5 + 1) / perUnit + thrown,
            reach: (pointSize * SPRITE_MOST * 0.5 + 1) / perUnit + thrown,
        }

        if (!this.sighting || !stillHolds(this.sighting, now)) {
            const slack = CULL_SLACK / sy
            const {right, up, back, limb, reach} = now
            const view: ChunkView = {toClip: m, back: [back.x, back.y, back.z], limb: limb + slack, reach: reach + slack}
            const shown = showingChunks(this.chunks, view, this.shown)
            const ranges = drawnRanges(this.chunks, this.shown, shown, MAX_DRAWS, LEAST_GAP)
            this.sighting = {...now, right, up, back, ranges, slack}
        }
        for (const {start, count} of this.sighting.ranges) geometry.addGroup(start, count, 0)
    }

    // Wherever they are: a ring is a handful of tiles, and the vertex shader drops those behind the earth.
    private ringRanges(): Span[] {
        if (!this.rings) {
            let count = 0
            this.ringed.forEach((tiles, chunk) => {
                if (tiles > 0) this.shown[count++] = chunk
            })
            this.rings = drawnRanges(this.chunks, this.shown, count, MAX_DRAWS, LEAST_GAP)
        }
        return this.rings
    }

    setOwners(changes: OwnerChange[]) {
        const tiles: number[] = []
        const flags: number[] = []
        for (const {tile, country} of changes) {
            const flag = country === undefined ? UNOWNED : flagNumbers.get(country)
            if (flag === undefined) {
                warnOnce(`No sprite region for country "${country}", leaving its tiles blank`)
                continue
            }
            tiles.push(tile)
            flags.push(flag)
        }

        this.patch(this.slots.flag, tiles, (values, at, i) => {
            if (this.compact) values[at] = flags[i]
            else values.set(PACKED.subarray(flags[i] * 2, flags[i] * 2 + 2), at)
        })

        tiles.forEach((tile, i) => {
            const texel = this.texels[tile - 1]
            this.tileValues[texel * 2] = flags[i]
            this.changedRows.add(Math.floor(texel / TILE_COLUMNS))
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

    setInfectedMarks(tiles: readonly number[], on: boolean) {
        this.mark(tiles, INFECTED_MARK, on)
    }

    setMineMarks(tiles: readonly number[], on: boolean) {
        this.mark(tiles, MINE_MARK, on)
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
        if (this.changedRows.size === 0) return
        const rows = Uint32Array.from(this.changedRows).sort()
        this.changedRows.clear()
        for (const {start, count} of joinSpans(spansOf(rows), MAX_ROW_UPLOADS)) {
            renderer.copyTextureToTexture(
                this.tileSource,
                this.tileData,
                new THREE.Box2(new THREE.Vector2(0, start), new THREE.Vector2(TILE_COLUMNS, start + count)),
                new THREE.Vector2(0, start),
            )
        }
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
        this.displayPoints.geometry.dispose()
        for (const material of this.displayPoints.material as THREE.Material[]) disposeMaterial(material)
        this.tileData.dispose()
        this.flagRectangles.dispose()
    }

    private mark(tiles: readonly number[], bit: number, on: boolean) {
        this.patch(this.slots.pulse, tiles, (values, at, i) => {
            const was = values[at]
            const is = on ? was | bit : was & ~bit
            values[at] = is
            if (!(was & RINGS) !== !(is & RINGS)) this.ring(tiles[i] - 1, (is & RINGS) !== 0)
        })
    }

    private ring(tile: number, on: boolean) {
        const chunk = chunkOf(this.chunks, tile)
        this.ringed[chunk] += on ? 1 : -1
        this.rings = undefined
    }

    private patch(slot: Slot, tiles: readonly number[], write: (values: THREE.TypedArray, at: number, i: number) => void) {
        if (tiles.length === 0) return

        const {attribute, offset, width} = slot
        const starts = new Float64Array(tiles.length)
        tiles.forEach((tile, i) => {
            const at = (tile - 1) * attribute.itemSize + offset
            write(attribute.array, at, i)
            starts[i] = at
        })

        const spans = joinSpans(spansOf(starts.sort(), width), MAX_UPDATE_RANGES, attribute.itemSize * JOINED_TILES)
        for (const {start, count} of spans) attribute.addUpdateRange(start, count)
        attribute.needsUpdate = true
    }
}

// The screen no larger, the sprites no wider, and the camera turned and moved by less than the slack.
function stillHolds(last: Sighting, now: Omit<Sighting, "ranges" | "slack">): boolean {
    const close = 1 - 1e-9
    if (now.sx < last.sx * close || now.sy < last.sy * close || now.limb * close > last.limb || now.reach * close > last.reach) return false
    return now.right.distanceTo(last.right) + Math.abs(now.tx - last.tx) <= last.slack
        && now.up.distanceTo(last.up) + Math.abs(now.ty - last.ty) <= last.slack
        && now.back.distanceTo(last.back) <= last.slack
}

// Only 32-bit floats: the white globe on Intel under Windows came with the compact layout's small integers.
function plainLayout(size: number): Layout {
    const flagRegion = new THREE.BufferAttribute(new Float32Array(size * 2), 2)
    const tileMarks = new THREE.BufferAttribute(new Float32Array(size * 4), 4)
    const at = (attribute: THREE.BufferAttribute, offset: number) => ({attribute, offset, width: 1})

    return {
        slots: {
            flag: {attribute: flagRegion, offset: 0, width: 2},
            landmass: at(tileMarks, 0),
            hover: at(tileMarks, 1),
            pulse: at(tileMarks, 2),
            shield: at(tileMarks, 3),
        },
        attributes: {flagRegion, tileMarks},
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
