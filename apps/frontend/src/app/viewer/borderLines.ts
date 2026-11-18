import * as THREE from "three"
import {flagPaint} from "./pointSize.ts"
import {type BoardScale, EARTH_1_SPACING} from "../../domain/boardScale.ts"
import {MapView} from "../../domain/displaySettings.ts"
import {type BorderLineData, decodeBorderLines} from "../../domain/borderLinesFile.ts"
import {LEAST_SLANT} from "../../domain/coarseOutline.ts"
import {type OutlineSet, type OutlineSets, outlineSetsOf, type Segments} from "../../domain/outlineSets.ts"
import {chunkCount} from "../../domain/tileChunks.ts"
import type {Span} from "../../domain/spans.ts"
import {ChunkSight, CULL_SLACK} from "./chunkSight.ts"
import {disposeMaterial} from "./scene.ts"
import type {OutlineAnswer, OutlineQuestion} from "./borderLinesWorker.ts"

import vertexShader from "./shaders/borderLine/vertex.glsl"
import fragmentShader from "./shaders/borderLine/fragment.glsl"

// tolerances: the coarse sets' (coarseTolerancesOf), built with the cell edges' chunks off the page.
export async function loadBorderLines(url: string, tolerances: readonly number[], signal?: AbortSignal): Promise<OutlineSets> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    const file = await response.arrayBuffer()
    try {
        return await inWorker(file, tolerances, signal)
    } catch (e) {
        if (signal?.aborted) throw e
        console.error(`could not build the border lines in a worker on ${navigator.userAgent}, building them on the page`, e)
        return outlineSetsOf(decodeBorderLines(file), tolerances)
    }
}

// Copied to the worker, not moved: the page builds from the same file if the worker fails.
function inWorker(file: ArrayBuffer, tolerances: readonly number[], signal?: AbortSignal): Promise<OutlineSets> {
    return new Promise((resolve, reject) => {
        signal?.throwIfAborted()
        const worker = new Worker(new URL("./borderLinesWorker.ts", import.meta.url), {type: "module"})
        const settle = (finish: () => void) => {
            worker.terminate()
            signal?.removeEventListener("abort", abort)
            finish()
        }
        const abort = () => settle(() => reject(signal?.reason))
        signal?.addEventListener("abort", abort, {once: true})
        worker.onmessage = ({data}: MessageEvent<OutlineAnswer>) => settle(() => {
            if ("cells" in data) resolve(data)
            else reject(new Error(data.error))
        })
        worker.onerror = (event) => settle(() => reject(new Error(event.message || "the border lines worker did not start")))
        const question: OutlineQuestion = {file, tolerances}
        worker.postMessage(question)
    })
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

export const HIGHLIGHT_WIDTH = 2.6

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

// The coarse sets the over and the highlight pass draw from orbit, in cell edges: how far a dropped corner may lie from
// its piece. A √2 step apart, so a screen of any height finds one near its pixel.
const COARSE = [0.5, Math.SQRT1_2, 1, Math.SQRT2, 2]

// In drawing-buffer pixels: how far the over and the highlight pass may draw from the cell edges.
export const TOLERANCE = 0.5

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

// The tolerance of each coarse set on the board, on the globe of radius 1.
export function coarseTolerancesOf(scale: BoardScale): number[] {
    const cellEdge = scale.length * EARTH_1_SPACING / Math.sqrt(3)
    return COARSE.map((edges) => edges * cellEdge)
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
    // A coarse set's: the cell edges each piece stands for, and their chord over their length.
    standsFor?: THREE.InstancedInterleavedBuffer
    draws: THREE.InstancedBufferGeometry[]
}

function piecesOf(lines: BorderLineData, {from, to}: Segments, standsFor?: Float32Array): Pieces {
    const corner = new THREE.BufferAttribute(new Float32Array([-1, -1, -1, 1, 1, -1, 1, 1]), 2)
    const index = new THREE.Uint16BufferAttribute([0, 1, 2, 2, 1, 3], 1)
    const pieces: Pieces = {
        from: new THREE.InstancedInterleavedBuffer(from, 3),
        to: new THREE.InstancedInterleavedBuffer(to, 3),
        sides: new THREE.InstancedInterleavedBuffer(pieceSides(lines), 2),
        standsFor: standsFor && new THREE.InstancedInterleavedBuffer(standsFor, 2),
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
    if (pieces.standsFor) geometry.setAttribute("standsFor", new THREE.InterleavedBufferAttribute(pieces.standsFor, 2, first * 2))
}

// The cell edges (tolerance 0) or a coarse set, each culled on its own.
type Level = {tolerance: number, pieces: Pieces, sight: ChunkSight, drawn: readonly Span[]}

function levelOf({lines, segments, chunks}: OutlineSet, tolerance: number, standsFor?: Float32Array): Level {
    const level: Level = {
        tolerance,
        pieces: piecesOf(lines, segments, standsFor),
        sight: new ChunkSight(chunks, new Uint32Array(chunkCount(chunks)), MAX_DRAWS, LEAST_GAP),
        drawn: [],
    }
    drawIn(level, [{start: 0, count: segments.from.length / 3}])
    return level
}

function drawIn(level: Level, ranges: readonly Span[]) {
    level.drawn = ranges
    ranges.forEach(({start, count}, at) => {
        startAt(level.pieces.draws[at], level.pieces, start)
        level.pieces.draws[at].instanceCount = count
    })
}

type Pass = {group: THREE.Group, material: THREE.ShaderMaterial, width: number, level: Level}

function passOf(level: Level, material: THREE.ShaderMaterial, width: number, renderOrder: number): Pass {
    const group = new THREE.Group()
    for (const geometry of level.pieces.draws) {
        const mesh = new THREE.Mesh(geometry, material)
        mesh.renderOrder = renderOrder
        mesh.frustumCulled = false
        group.add(mesh)
    }
    return {group, material, width, level}
}

// A pass drawn from orbit: a coarse set inks each pixel as the cell edges' quads laid over one another did.
type OrbitPass = Pass & {stacked: THREE.ShaderMaterial}

function orbitPassOf(level: Level, material: THREE.ShaderMaterial, width: number, renderOrder: number): OrbitPass {
    const stacked = new THREE.ShaderMaterial({
        defines: {...material.defines, STACKED: true},
        // The plain material's own, so what is set on one is set on both.
        uniforms: material.uniforms,
        vertexShader,
        fragmentShader,
        transparent: true,
        side: THREE.DoubleSide,
        depthWrite: false,
    })
    return {...passOf(level, material, width, renderOrder), stacked}
}

function drawFrom(pass: OrbitPass, level: Level) {
    if (pass.level === level) return
    pass.level = level
    for (const [at, mesh] of (pass.group.children as THREE.Mesh[]).entries()) {
        mesh.geometry = level.pieces.draws[at]
        mesh.material = level.pieces.standsFor ? pass.stacked : pass.material
    }
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
export function createBorderLines(sets: OutlineSets, scale: BoardScale): BorderLines {
    const fine = levelOf(sets.cells, 0)
    const coarse = sets.coarse.map((set) => levelOf(set, set.tolerance, set.standsFor))
    const levels = [fine, ...coarse]
    const aimed = aimedTexture(sets.cells.lines)

    // After the tiles, so their depth hides the under pass; made in drawing order, as three draws one renderOrder by id.
    const over = orbitPassOf(fine, lineMaterial({lift: OVER, width: ORBIT_WIDTH, colour: ORBIT_COLOUR}), ORBIT_WIDTH, 1)
    const under = passOf(fine, lineMaterial({lift: UNDER, width: WIDTH, colour: COLOUR}), WIDTH, 1)
    const highlight = orbitPassOf(fine, lineMaterial({
        lift: OVER, width: HIGHLIGHT_WIDTH, colour: HIGHLIGHT_COLOUR,
        defines: {HIGHLIGHT: true}, uniforms: {landmass: {value: 0}},
    }), HIGHLIGHT_WIDTH, 2)
    const mission = passOf(fine, lineMaterial({
        lift: OVER, width: MISSION_WIDTH, colour: MISSION_COLOUR, ink: MISSION_INK,
        defines: {MISSION: true}, uniforms: {aimed: {value: aimed}},
    }), MISSION_WIDTH, 2)
    highlight.group.visible = false
    mission.group.visible = false
    const passes = [over, under, highlight, mission]

    const object = new THREE.Group()
    object.add(...passes.map(({group}) => group))

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

            // The camera is orthographic, so nothing on the globe shows longer than it is: a set's dropped corners stay
            // within its tolerance times perUnit of the cell edges, anywhere on screen.
            let orbit = fine
            for (const level of coarse) if (level.tolerance * OVER * perUnit <= TOLERANCE) orbit = level
            drawFrom(over, orbit)
            drawFrom(highlight, orbit)

            const halfWidth = halfWidthOf(ratio, WIDEST)
            const slack = Math.min(CULL_SLACK / sy, MOST_SLACK)
            for (const level of orbit === fine ? [fine] : [fine, orbit]) {
                // A coarse piece's quad is wider than its line, as much as its cell edges slant across it.
                const across = level === fine ? halfWidth : halfWidth / LEAST_SLANT
                // The vertex shader's own drop, and the quad's corners, which reach past its ends.
                const limb = limbOf(OVER) + (across + 1) / Math.max(perUnit, 1)
                const reach = (Math.hypot(halfWidth, across) + 1) / perUnit + FURTHEST_LIFT
                const ranges = level.sight.rangesFor(camera, m, limb, reach, slack)
                if (ranges !== level.drawn) drawIn(level, ranges)
            }
            for (const {group, level} of passes) {
                group.children.forEach((mesh, at) => {
                    mesh.visible = at < level.drawn.length
                })
            }
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
            for (const {pieces} of levels) for (const geometry of pieces.draws) geometry.dispose()
            for (const {material} of passes) disposeMaterial(material)
            for (const {stacked} of [over, highlight]) stacked.dispose()
            aimed.dispose()
        },
    }
}
