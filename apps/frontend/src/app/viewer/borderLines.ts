import * as THREE from "three"
import {flagPaint} from "./pointSize.ts"
import type {BoardScale} from "../../domain/boardScale.ts"
import {MapView} from "../../domain/displaySettings.ts"
import {type BorderLineData, decodeBorderLines} from "../../domain/borderLinesFile.ts"
import {CHUNK_MOST, CHUNK_REACH, chunkCount, chunksOf, type TileChunks} from "../../domain/tileChunks.ts"
import type {Span} from "../../domain/spans.ts"
import {ChunkSight, CULL_SLACK} from "./chunkSight.ts"
import {disposeMaterial} from "./scene.ts"

import vertexShader from "./shaders/borderLine/vertex.glsl"
import fragmentShader from "./shaders/borderLine/fragment.glsl"

export async function loadBorderLines(url: string, signal?: AbortSignal): Promise<BorderLineData> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    return decodeBorderLines(await response.arrayBuffer())
}

// A piece is one cell edge, corner to corner: a run of n corners is n - 1 pieces.
export type Segments = {from: Int16Array, to: Int16Array}

export function outlineSegments(data: BorderLineData): Segments {
    const pieces = data.corners.length / 3 - data.runs.length
    const from = new Int16Array(pieces * 3)
    const to = new Int16Array(pieces * 3)

    let at = 0, piece = 0
    for (const corners of data.runs) {
        from.set(data.corners.subarray(at * 3, (at + corners - 1) * 3), piece * 3)
        to.set(data.corners.subarray((at + 1) * 3, (at + corners) * 3), piece * 3)
        piece += corners - 1
        at += corners
    }
    return {from, to}
}

// The two landmasses each piece runs between, the sea being 0.
export function pieceSides(data: BorderLineData): Uint16Array {
    const sides = new Uint16Array((data.corners.length / 3 - data.runs.length) * 2)
    let piece = 0
    data.runs.forEach((corners, run) => {
        const pair = data.sides.subarray(run * 2, run * 2 + 2)
        for (let i = 1; i < corners; i++, piece++) sides.set(pair, piece * 2)
    })
    return sides
}

export const WIDTH = 1.3

export const ORBIT_WIDTH = 0.25

const HIGHLIGHT_WIDTH = 2.6

const FEATHER = 1

export const OVER = 1.0005
export const UNDER = 0.9995

const EARTH = 0.999

export function limbOf(lift: number): number {
    return Math.sin(Math.acos(Math.min(EARTH / lift, 1)))
}

const COLOUR = new THREE.Color(0.25, 0.25, 0.25)

// Over a painted flag the line darkens what is under it rather than drawing grey across it.
const ORBIT_COLOUR = new THREE.Color(0, 0, 0)
export const ORBIT_INK = 0.7

const HIGHLIGHT_COLOUR = new THREE.Color(1, 1, 1)

// Must match MISSION in shaders/display/fragment.glsl and --mission in tokens.css: one colour for a mission's place.
const MISSION_COLOUR = new THREE.Color(0.784, 0.341, 0.298)
const MISSION_WIDTH = 2
const MISSION_INK = 0.9

// Must match AIMED_COLUMNS in shaders/borderLine/vertex.glsl.
const AIMED_COLUMNS = 1024

export function halfWidthOf(pixelRatio: number, width = WIDTH): number {
    return width * pixelRatio / 2 + FEATHER
}

// Every pass's at most, so the cull keeps any piece one of them would draw.
const WIDEST = Math.max(WIDTH, ORBIT_WIDTH, HIGHLIGHT_WIDTH, MISSION_WIDTH)
const FURTHEST_LIFT = Math.max(OVER - 1, 1 - UNDER)

const MAX_DRAWS = 8
// In pieces.
const LEAST_GAP = 512

// Of the globe's radius: zoomed out, a quarter of the screen would keep most of the far side.
export const MOST_SLACK = 0.05

export type BorderLines = {
    object: THREE.Object3D
    update(zoom: number, width: number, height: number, pixelRatio: number, view: MapView): void
    cull(camera: THREE.Camera): void
    setHighlight(landmass: number): void
    setAimed(landmasses: ReadonlySet<number>): void
    dispose(): void
}

// A cap holds both ends of each of its pieces.
export function outlineChunksOf({from, to}: Segments): TileChunks {
    const pieces = from.length / 3
    const ends = new Int16Array(pieces * 6)
    for (let piece = 0; piece < pieces; piece++) {
        ends.set(from.subarray(piece * 3, piece * 3 + 3), piece * 6)
        ends.set(to.subarray(piece * 3, piece * 3 + 3), piece * 6 + 3)
    }
    return chunksOf(ends, CHUNK_REACH, CHUNK_MOST, 2)
}

function aimedTexture(data: BorderLineData): THREE.DataTexture {
    const landmasses = data.sides.reduce((most, side) => Math.max(most, side), 0) + 1
    const rows = Math.ceil(landmasses / AIMED_COLUMNS)
    const texture = new THREE.DataTexture(new Uint8Array(AIMED_COLUMNS * rows), AIMED_COLUMNS, rows, THREE.RedFormat)
    texture.needsUpdate = true
    return texture
}

type Pieces = {
    from: THREE.InstancedInterleavedBuffer
    to: THREE.InstancedInterleavedBuffer
    sides: THREE.InstancedInterleavedBuffer
    draws: THREE.InstancedBufferGeometry[]
}

function piecesOf(data: BorderLineData, {from, to}: Segments): Pieces {
    const corner = new THREE.BufferAttribute(new Float32Array([-1, -1, -1, 1, 1, -1, 1, 1]), 2)
    const index = new THREE.Uint16BufferAttribute([0, 1, 2, 2, 1, 3], 1)
    const pieces: Pieces = {
        from: new THREE.InstancedInterleavedBuffer(from, 3),
        to: new THREE.InstancedInterleavedBuffer(to, 3),
        sides: new THREE.InstancedInterleavedBuffer(pieceSides(data), 2),
        draws: [],
    }
    for (let draw = 0; draw < MAX_DRAWS; draw++) {
        const geometry = new THREE.InstancedBufferGeometry()
        geometry.setAttribute("corner", corner)
        geometry.setIndex(index)
        startAt(geometry, pieces, 0)
        geometry.instanceCount = 0
        pieces.draws.push(geometry)
    }
    return pieces
}

// WebGL2 draws no range of instances, so a draw's attributes start at its range's first piece.
function startAt(geometry: THREE.InstancedBufferGeometry, pieces: Pieces, first: number) {
    const from = geometry.getAttribute("from") as THREE.InterleavedBufferAttribute | undefined
    if (from?.offset === first * 3) return
    geometry.setAttribute("from", new THREE.InterleavedBufferAttribute(pieces.from, 3, first * 3, true))
    geometry.setAttribute("to", new THREE.InterleavedBufferAttribute(pieces.to, 3, first * 3, true))
    geometry.setAttribute("sides", new THREE.InterleavedBufferAttribute(pieces.sides, 2, first * 2))
}

type Pass = {group: THREE.Group, material: THREE.ShaderMaterial, width: number}

function passOf(pieces: Pieces, material: THREE.ShaderMaterial, width: number, renderOrder: number): Pass {
    const group = new THREE.Group()
    for (const geometry of pieces.draws) {
        const mesh = new THREE.Mesh(geometry, material)
        mesh.renderOrder = renderOrder
        mesh.frustumCulled = false
        group.add(mesh)
    }
    return {group, material, width}
}

type Line = {
    lift: number
    width: number
    colour: THREE.Color
    ink?: number
    defines?: Record<string, boolean>
    uniforms?: Record<string, THREE.IUniform>
}

function lineMaterial({lift, width, colour, ink = 1, defines = {}, uniforms = {}}: Line): THREE.ShaderMaterial {
    return new THREE.ShaderMaterial({
        defines,
        uniforms: {
            halfViewport: {value: new THREE.Vector2(1, 1)},
            halfWidth: {value: halfWidthOf(1, width)},
            lift: {value: lift},
            limb: {value: limbOf(lift)},
            colour: {value: colour},
            ink: {value: ink},
            ...uniforms,
        },
        vertexShader,
        fragmentShader,
        transparent: true,
        side: THREE.DoubleSide,
        depthWrite: false,
    })
}

// scale: the board the outline is traced on, whose tiles the painted flags hand over to.
export function createBorderLines(data: BorderLineData, scale: BoardScale): BorderLines {
    const segments = outlineSegments(data)
    const chunks = outlineChunksOf(segments)
    const pieces = piecesOf(data, segments)
    const aimed = aimedTexture(data)

    // After the tiles, so their depth hides the under pass; made in drawing order, as three draws one renderOrder by id.
    const over = passOf(pieces, lineMaterial({lift: OVER, width: ORBIT_WIDTH, colour: ORBIT_COLOUR}), ORBIT_WIDTH, 1)
    const under = passOf(pieces, lineMaterial({lift: UNDER, width: WIDTH, colour: COLOUR}), WIDTH, 1)
    const highlight = passOf(pieces, lineMaterial({
        lift: OVER, width: HIGHLIGHT_WIDTH, colour: HIGHLIGHT_COLOUR,
        defines: {HIGHLIGHT: true}, uniforms: {landmass: {value: 0}},
    }), HIGHLIGHT_WIDTH, 2)
    const mission = passOf(pieces, lineMaterial({
        lift: OVER, width: MISSION_WIDTH, colour: MISSION_COLOUR, ink: MISSION_INK,
        defines: {MISSION: true}, uniforms: {aimed: {value: aimed}},
    }), MISSION_WIDTH, 2)
    highlight.group.visible = false
    mission.group.visible = false
    const passes = [over, under, highlight, mission]

    const object = new THREE.Group()
    object.add(...passes.map(({group}) => group))

    let drawn: readonly Span[] = []
    const draw = (ranges: readonly Span[]) => {
        drawn = ranges
        ranges.forEach(({start, count}, at) => {
            startAt(pieces.draws[at], pieces, start)
            pieces.draws[at].instanceCount = count
        })
        for (const {group} of passes) {
            group.children.forEach((mesh, at) => {
                mesh.visible = at < ranges.length
            })
        }
    }
    draw([{start: 0, count: segments.from.length / 3}])

    const sight = new ChunkSight(chunks, new Uint32Array(chunkCount(chunks)), MAX_DRAWS, LEAST_GAP)
    const toClip = new THREE.Matrix4()
    let halfHeight = 1
    let ratio = 1

    return {
        object,
        update(zoom: number, width: number, height: number, pixelRatio: number, view: MapView) {
            halfHeight = height / 2
            ratio = pixelRatio
            for (const {material, width: line} of passes) {
                material.uniforms.halfViewport.value.set(width / 2, height / 2)
                material.uniforms.halfWidth.value = halfWidthOf(pixelRatio, line)
            }

            const paint = flagPaint(zoom, height / pixelRatio, view, scale)
            over.material.uniforms.ink.value = paint * ORBIT_INK
            over.group.visible = paint > 0
            under.group.visible = paint < 1
        },
        cull(camera: THREE.Camera) {
            camera.updateMatrixWorld()
            toClip.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)
            const m = toClip.elements
            const sy = Math.hypot(m[1], m[5], m[9])
            const perUnit = halfHeight * sy
            const halfWidth = halfWidthOf(ratio, WIDEST)
            // The vertex shader's own drop, and the quad's corners, which reach past its ends by √2 half-widths.
            const limb = limbOf(OVER) + (halfWidth + 1) / Math.max(perUnit, 1)
            const reach = (Math.SQRT2 * halfWidth + 1) / perUnit + FURTHEST_LIFT
            const ranges = sight.rangesFor(camera, m, limb, reach, Math.min(CULL_SLACK / sy, MOST_SLACK))
            if (ranges !== drawn) draw(ranges)
        },
        setHighlight(landmass: number) {
            highlight.material.uniforms.landmass.value = landmass
            highlight.group.visible = landmass > 0
        },
        setAimed(landmasses: ReadonlySet<number>) {
            const marks = aimed.image.data as Uint8Array
            marks.fill(0)
            for (const landmass of landmasses) marks[landmass] = 255
            aimed.needsUpdate = true
            mission.group.visible = landmasses.size > 0
        },
        dispose() {
            for (const geometry of pieces.draws) geometry.dispose()
            for (const {material} of passes) disposeMaterial(material)
            aimed.dispose()
        },
    }
}
