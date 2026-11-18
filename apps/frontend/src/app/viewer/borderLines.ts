import * as THREE from "three"
import {flagPaint, inOrbit, tilePointSize} from "./pointSize.ts"
import {type BoardScale, EARTH_1_SPACING} from "../../domain/boardScale.ts"
import type {Divisions, MapView} from "../../domain/displaySettings.ts"
import {type BorderLineData, decodeBorderLines} from "../../domain/borderLinesFile.ts"
import {LEAST_SLANT} from "../../domain/coarseOutline.ts"
import {decodeProvinceLines} from "../../domain/provinceLinesFile.ts"
import {
    type OutlineSet,
    type OutlineSets,
    outlineSetsOf,
    type ProvinceOutline,
    provinceOutlineOf,
    type Segments,
} from "../../domain/outlineSets.ts"
import {chunkCount} from "../../domain/tileChunks.ts"
import type {Span} from "../../domain/spans.ts"
import {ChunkSight, CULL_SLACK} from "./chunkSight.ts"
import {disposeMaterial} from "./scene.ts"
import type {OutlineAnswer, OutlineQuestion} from "./borderLinesWorker.ts"

import vertexShader from "./shaders/borderLine/vertex.glsl"
import fragmentShader from "./shaders/borderLine/fragment.glsl"

// tolerances: the coarse sets' (coarseTolerancesOf), built with the cell edges' chunks off the page.
export async function loadBorderLines(url: string, tolerances: readonly number[], signal?: AbortSignal): Promise<OutlineSets> {
    const file = await fileAt(url, signal)
    try {
        const answer = await inWorker({kind: "countries", file, tolerances}, signal)
        if (answer.kind === "countries") return answer.sets
        throw new Error(`the worker answered ${answer.kind}`)
    } catch (e) {
        if (signal?.aborted) throw e
        console.error(`could not build the border lines in a worker on ${navigator.userAgent}, building them on the page`, e)
        return outlineSetsOf(decodeBorderLines(file), tolerances)
    }
}

// A board's province lines, their sets built as the countries' are.
export async function loadProvinceLines(url: string, tolerances: readonly number[], signal?: AbortSignal): Promise<ProvinceOutline> {
    const file = await fileAt(url, signal)
    try {
        const answer = await inWorker({kind: "provinces", file, tolerances}, signal)
        if (answer.kind === "provinces") return answer.outline
        throw new Error(`the worker answered ${answer.kind}`)
    } catch (e) {
        if (signal?.aborted) throw e
        console.error(`could not build the province lines in a worker on ${navigator.userAgent}, building them on the page`, e)
        return provinceOutlineOf(decodeProvinceLines(file), tolerances)
    }
}

async function fileAt(url: string, signal?: AbortSignal): Promise<ArrayBuffer> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    return response.arrayBuffer()
}

// Copied to the worker, not moved: the page builds from the same file if the worker fails.
function inWorker(question: OutlineQuestion, signal?: AbortSignal): Promise<OutlineAnswer> {
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
            if (data.kind === "error") reject(new Error(data.error))
            else resolve(data)
        })
        worker.onerror = (event) => settle(() => reject(new Error(event.message || "the border lines worker did not start")))
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

export const COLOUR = new THREE.Color(0.25, 0.25, 0.25)

// Over a painted flag the line darkens what is under it rather than drawing grey across it.
const ORBIT_COLOUR = new THREE.Color(0, 0, 0)
export const ORBIT_INK = 0.7

const HIGHLIGHT_COLOUR = new THREE.Color(1, 1, 1)

// Between two provinces of one country. Under the tiles, as they hand over from the painted flags, lighter than the
// countries' line; pushed in, as dark as theirs and dashed, so a province reads as clearly as a country and is not
// taken for one. Over a painted flag a light line rather than a dark one.
export const PROVINCE_COLOUR = new THREE.Color(0.6, 0.6, 0.6)
const PROVINCE_ORBIT_COLOUR = new THREE.Color(0.85, 0.85, 0.85)
export const PROVINCE_ORBIT_INK = 0.55

// A tile's disc on screen, in CSS pixels, from which the province line darkens, and at which it is the countries' and
// dashed. The tiles take the frame at 8 (pointSize.ts); the ground's own cell edges darken from 10 to 18 cells wide.
export const PROVINCE_DARK_FROM = 10
export const PROVINCE_DARK_UNTIL = 20

// In CSS pixels: a dash and the gap after it, which a piece rounds to a whole number of.
export const PROVINCE_DASH = 8
export const PROVINCE_GAP = 4

// 0 the light solid line, 1 the countries' ink, dashed.
export function provinceDarkOf(tileSize: number): number {
    return Math.min(1, Math.max(0, (tileSize - PROVINCE_DARK_FROM) / (PROVINCE_DARK_UNTIL - PROVINCE_DARK_FROM)))
}

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
    update(zoom: number, width: number, height: number, pixelRatio: number, view: MapView, divisions: Divisions): void
    cull(camera: THREE.Camera): void
    // The landmass or the province lit, by the divisions it is one of; 0 for none.
    setHighlight(divisions: Divisions, id: number): void
    setAimed(landmasses: ReadonlySet<number>): void
    // The lines between two provinces of one country, and the provinces either side of each of the countries' cell edges.
    takeProvinces(outline: ProvinceOutline): void
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
    // The provinces either side of each piece, when the pieces are the countries' cell edges and the board has provinces.
    provinces?: THREE.InstancedInterleavedBuffer
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
    if (pieces.provinces) geometry.setAttribute("provinces", new THREE.InterleavedBufferAttribute(pieces.provinces, 2, first * 2))
}

// The provinces either side of each piece, as a new attribute of every draw, from where its range starts.
function takeSides({pieces}: Level, sides: Uint16Array) {
    pieces.provinces = new THREE.InstancedInterleavedBuffer(sides, 2)
    for (const geometry of pieces.draws) {
        const first = (geometry.getAttribute("from") as THREE.InterleavedBufferAttribute).offset / 3
        geometry.setAttribute("provinces", new THREE.InterleavedBufferAttribute(pieces.provinces, 2, first * 2))
    }
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

function highlightOf(defines: Record<string, boolean> = {}): THREE.ShaderMaterial {
    return lineMaterial({
        lift: OVER, width: HIGHLIGHT_WIDTH, colour: HIGHLIGHT_COLOUR,
        defines: {HIGHLIGHT: true, ...defines}, uniforms: {highlit: {value: 0}},
    })
}

function light({group, material}: Pass, id: number) {
    material.uniforms.highlit.value = id
    group.visible = id > 0
}

// One kind of line: its cell edges and its coarse sets, the same tolerances for every kind.
type Kind = {fine: Level, coarse: Level[]}

function kindOf(sets: OutlineSets): Kind {
    return {fine: levelOf(sets.cells, 0), coarse: sets.coarse.map((set) => levelOf(set, set.tolerance, set.standsFor))}
}

// The province lines' passes, and the countries' cell edges lit by province.
type ProvinceLines = {kind: Kind, over: OrbitPass, under: Pass, highlight: OrbitPass, countryHighlight: Pass}

// scale: the board the outline is traced on, whose tiles the painted flags hand over to.
export function createBorderLines(sets: OutlineSets, scale: BoardScale): BorderLines {
    const countries = kindOf(sets)
    const {fine, coarse} = countries
    const levels = [fine, ...coarse]
    const aimed = aimedTexture(sets.cells.lines)

    // After the tiles, so their depth hides the under pass; made in drawing order, as three draws one renderOrder by id.
    const over = orbitPassOf(fine, lineMaterial({lift: OVER, width: ORBIT_WIDTH, colour: ORBIT_COLOUR}), ORBIT_WIDTH, 1)
    const under = passOf(fine, lineMaterial({lift: UNDER, width: WIDTH, colour: COLOUR}), WIDTH, 1)
    const highlight = orbitPassOf(fine, highlightOf(), HIGHLIGHT_WIDTH, 2)
    const mission = passOf(fine, lineMaterial({
        lift: OVER, width: MISSION_WIDTH, colour: MISSION_COLOUR, ink: MISSION_INK,
        defines: {MISSION: true}, uniforms: {aimed: {value: aimed}},
    }), MISSION_WIDTH, 2)
    highlight.group.visible = false
    mission.group.visible = false
    const passes: Pass[] = [over, under, highlight, mission]
    const orbitPasses: OrbitPass[] = [over, highlight]

    const object = new THREE.Group()
    object.add(...passes.map(({group}) => group))

    // Once the board's province lines are read.
    let provinces: ProvinceLines | undefined
    let lit: {divisions: Divisions, id: number} = {divisions: "countries", id: 0}
    let shown = {zoom: 1, height: 1, ratio: 1, view: "split" as MapView, divisions: "countries" as Divisions}
    const halfViewport = new THREE.Vector2(1, 1)

    const toClip = new THREE.Matrix4()

    const size = (pass: Pass) => {
        pass.material.uniforms.halfViewport.value.copy(halfViewport)
        pass.material.uniforms.halfWidth.value = halfWidthOf(shown.ratio, pass.width)
    }

    const showLines = () => {
        const {zoom, height, ratio, view, divisions} = shown
        const paint = flagPaint(zoom, height / ratio, view, scale)
        const dark = provinceDarkOf(tilePointSize(zoom, height / ratio, scale))
        over.material.uniforms.ink.value = paint * ORBIT_INK
        over.group.visible = paint > 0
        under.group.visible = paint < 1
        if (!provinces) return
        // From orbit only the provinces' divisions draw them; pushed in, every one does.
        const drawn = divisions === "provinces" || !inOrbit(zoom, height / ratio, scale)
        provinces.over.material.uniforms.ink.value = paint * PROVINCE_ORBIT_INK
        provinces.over.group.visible = divisions === "provinces" && paint > 0
        provinces.under.group.visible = drawn && paint < 1
        const {uniforms} = provinces.under.material
        uniforms.colour.value.lerpColors(PROVINCE_COLOUR, COLOUR, dark)
        uniforms.dashed.value = dark
        uniforms.dashPeriod.value = (PROVINCE_DASH + PROVINCE_GAP) * ratio
        uniforms.dashHalf.value = PROVINCE_DASH * ratio / 2
    }

    const showHighlight = () => {
        const {divisions, id} = lit
        light(highlight, divisions === "countries" ? id : 0)
        if (!provinces) return
        light(provinces.highlight, divisions === "provinces" ? id : 0)
        light(provinces.countryHighlight, divisions === "provinces" ? id : 0)
    }

    return {
        object,
        update(zoom: number, width: number, height: number, pixelRatio: number, view: MapView, divisions: Divisions) {
            halfViewport.set(width / 2, height / 2)
            shown = {zoom, height, ratio: pixelRatio, view, divisions}
            for (const pass of passes) size(pass)
            showLines()
        },
        cull(camera: THREE.Camera) {
            camera.updateMatrixWorld()
            toClip.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)
            const m = toClip.elements
            const sy = Math.hypot(m[1], m[5], m[9])
            const perUnit = halfViewport.y * sy

            // The camera is orthographic, so nothing on the globe shows longer than it is: a set's dropped corners stay
            // within its tolerance times perUnit of the cell edges, anywhere on screen. Every kind of line has the same
            // tolerances, so one choice holds for all.
            let coarsest = -1
            coarse.forEach((level, at) => {
                if (level.tolerance * OVER * perUnit <= TOLERANCE) coarsest = at
            })
            const orbitOf = (kind: Kind) => coarsest < 0 ? kind.fine : kind.coarse[coarsest]
            drawFrom(over, orbitOf(countries))
            drawFrom(highlight, orbitOf(countries))
            if (provinces) {
                drawFrom(provinces.over, orbitOf(provinces.kind))
                drawFrom(provinces.highlight, orbitOf(provinces.kind))
            }

            const halfWidth = halfWidthOf(shown.ratio, WIDEST)
            const slack = Math.min(CULL_SLACK / sy, MOST_SLACK)
            const kinds = provinces ? [countries, provinces.kind] : [countries]
            for (const kind of kinds) {
                const orbit = orbitOf(kind)
                for (const level of orbit === kind.fine ? [kind.fine] : [kind.fine, orbit]) {
                    // A coarse piece's quad is wider than its line, as much as its cell edges slant across it.
                    const across = level === kind.fine ? halfWidth : halfWidth / LEAST_SLANT
                    // The vertex shader's own drop, and the quad's corners, which reach past its ends.
                    const limb = limbOf(OVER) + (across + 1) / Math.max(perUnit, 1)
                    const reach = (Math.hypot(halfWidth, across) + 1) / perUnit + FURTHEST_LIFT
                    const ranges = level.sight.rangesFor(camera, m, limb, reach, slack)
                    if (ranges !== level.drawn) drawIn(level, ranges)
                }
            }
            for (const {group, level} of passes) {
                group.children.forEach((mesh, at) => {
                    mesh.visible = at < level.drawn.length
                })
            }
        },
        setHighlight(divisions: Divisions, id: number) {
            lit = {divisions, id}
            showHighlight()
        },
        setAimed(landmasses: ReadonlySet<number>) {
            const marks = aimed.image.data as Uint8Array
            marks.fill(0)
            for (const landmass of landmasses) marks[landmass] = 255
            aimed.needsUpdate = true
            mission.group.visible = landmasses.size > 0
        },
        takeProvinces(outline: ProvinceOutline) {
            if (provinces) return
            // A coarse piece of the coast can run past where a province ends: a province's stretch of it is lit cell
            // edge by cell edge.
            takeSides(fine, outline.countrySides)
            const kind = kindOf(outline.sets)
            levels.push(kind.fine, ...kind.coarse)
            const added = {
                countryHighlight: passOf(fine, highlightOf({PROVINCES: true}), HIGHLIGHT_WIDTH, 2),
                over: orbitPassOf(kind.fine, lineMaterial({
                    lift: OVER, width: ORBIT_WIDTH, colour: PROVINCE_ORBIT_COLOUR, ink: 0,
                }), ORBIT_WIDTH, 1),
                under: passOf(kind.fine, lineMaterial({
                    lift: UNDER, width: WIDTH, colour: PROVINCE_COLOUR.clone(), defines: {DASHED: true},
                    uniforms: {dashed: {value: 0}, dashPeriod: {value: 1}, dashHalf: {value: 1}},
                }), WIDTH, 1),
                highlight: orbitPassOf(kind.fine, highlightOf(), HIGHLIGHT_WIDTH, 2),
            }
            provinces = {kind, ...added}
            const more = [added.countryHighlight, added.over, added.under, added.highlight]
            passes.push(...more)
            orbitPasses.push(added.over, added.highlight)
            for (const pass of more) {
                pass.group.visible = false
                object.add(pass.group)
                size(pass)
            }
            showLines()
            showHighlight()
        },
        dispose() {
            for (const {pieces} of levels) for (const geometry of pieces.draws) geometry.dispose()
            for (const {material} of passes) disposeMaterial(material)
            for (const {stacked} of orbitPasses) stacked.dispose()
            aimed.dispose()
        },
    }
}
