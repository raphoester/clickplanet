import * as THREE from "three"
import {flagPaint} from "./pointSize.ts"
import {MapView} from "../../domain/displaySettings.ts"
import {disposeMaterial} from "./scene.ts"

import vertexShader from "./shaders/borderLine/vertex.glsl"
import fragmentShader from "./shaders/borderLine/fragment.glsl"

export type BorderLineData = {
    runs: Uint32Array
    corners: Int16Array
}

const MAGIC = "CPBL"
const VERSION = 1

export function decodeBorderLines(buffer: ArrayBuffer): BorderLineData {
    const head = new DataView(buffer)
    const magic = String.fromCharCode(head.getUint8(0), head.getUint8(1), head.getUint8(2), head.getUint8(3))
    if (magic !== MAGIC) throw new Error(`not a border-line blob: magic "${magic}"`)
    const version = head.getUint32(4, true)
    if (version !== VERSION) throw new Error(`border-line blob version ${version}, expected ${VERSION}`)

    const runCount = head.getUint32(8, true)
    const cornerCount = head.getUint32(12, true)
    const runs = new Uint32Array(buffer, 16, runCount)
    const corners = new Int16Array(buffer, 16 + runCount * 4, cornerCount * 3)
    return {runs, corners}
}

export async function loadBorderLines(url: string, signal?: AbortSignal): Promise<BorderLineData> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    return decodeBorderLines(await response.arrayBuffer())
}

export const SAMPLES = 4

export const COARSE_SAMPLES = 2

type Run = {at: number, controls: number, closed: boolean}

function runsOf(data: BorderLineData): Run[] {
    const runs: Run[] = []
    let at = 0
    for (const corners of data.runs) {
        const closed = corners > 2 && sameCorner(data.corners, at, at + corners - 1)
        runs.push({at, controls: closed ? corners - 1 : corners, closed})
        at += corners
    }
    return runs
}

function sameCorner(corners: Int16Array, one: number, other: number): boolean {
    for (let axis = 0; axis < 3; axis++) {
        if (corners[one * 3 + axis] !== corners[other * 3 + axis]) return false
    }
    return true
}

function control(run: Run, step: number): number {
    const {at, controls, closed} = run
    if (closed) return at + ((step % controls) + controls) % controls
    return at + Math.min(Math.max(step, 0), controls - 1)
}

export function outlineSegments(data: BorderLineData, samples = SAMPLES): {from: Int16Array, to: Int16Array} {
    const runs = runsOf(data)

    let pieces = 0
    for (const run of runs) pieces += run.controls * samples

    const from = new Int16Array(pieces * 3)
    const to = new Int16Array(pieces * 3)

    let piece = 0
    for (const run of runs) {
        for (let step = 0; step < run.controls; step++) {
            const before = control(run, step - 1)
            const on = control(run, step)
            const after = control(run, step + 1)
            for (let cut = 0; cut < samples; cut++) {
                spline(data.corners, before, on, after, cut / samples, from, piece)
                spline(data.corners, before, on, after, (cut + 1) / samples, to, piece)
                piece++
            }
        }
    }

    return {from, to}
}

function spline(
    corners: Int16Array, before: number, on: number, after: number,
    t: number, into: Int16Array, piece: number,
) {
    const head = 0.5 * (1 - t) * (1 - t)
    const middle = 0.5 + t - t * t
    const tail = 0.5 * t * t
    for (let axis = 0; axis < 3; axis++) {
        into[piece * 3 + axis] = Math.round(
            head * corners[before * 3 + axis]
            + middle * corners[on * 3 + axis]
            + tail * corners[after * 3 + axis],
        )
    }
}

export const WIDTH = 1.3

const FEATHER = 1

export const OVER = 1.0005
export const UNDER = 0.9995

const EARTH = 0.999

export function limbOf(lift: number): number {
    return Math.sin(Math.acos(Math.min(EARTH / lift, 1)))
}

const COLOUR = new THREE.Color(0.25, 0.25, 0.25)

export function halfWidthOf(pixelRatio: number): number {
    return WIDTH * pixelRatio / 2 + FEATHER
}

export type BorderLines = {
    object: THREE.Object3D
    update(zoom: number, width: number, height: number, pixelRatio: number, view: MapView): void
    dispose(): void
}

function outlineGeometry(data: BorderLineData, samples: number): THREE.InstancedBufferGeometry {
    const {from, to} = outlineSegments(data, samples)

    const geometry = new THREE.InstancedBufferGeometry()
    geometry.setAttribute("corner", new THREE.BufferAttribute(
        new Float32Array([-1, -1, -1, 1, 1, -1, 1, 1]), 2,
    ))
    geometry.setIndex([0, 1, 2, 2, 1, 3])
    geometry.setAttribute("from", new THREE.InstancedBufferAttribute(from, 3, true))
    geometry.setAttribute("to", new THREE.InstancedBufferAttribute(to, 3, true))
    geometry.instanceCount = from.length / 3

    return geometry
}

export function createBorderLines(data: BorderLineData): BorderLines {
    const geometries = [
        outlineGeometry(data, COARSE_SAMPLES),
        outlineGeometry(data, SAMPLES),
    ]

    const passes = [OVER, UNDER].map((lift, at) => {
        const material = new THREE.ShaderMaterial({
            uniforms: {
                halfViewport: {value: new THREE.Vector2(1, 1)},
                halfWidth: {value: halfWidthOf(1)},
                lift: {value: lift},
                limb: {value: limbOf(lift)},
                colour: {value: COLOUR},
                ink: {value: 1},
            },
            vertexShader,
            fragmentShader,
            transparent: true,
            side: THREE.DoubleSide,
            depthWrite: false,
        })
        const mesh = new THREE.Mesh(geometries[at], material)
        // After the tiles, so their depth already hides the under pass where they cover it.
        mesh.renderOrder = 1
        mesh.frustumCulled = false
        return mesh
    })
    const [over, under] = passes

    const object = new THREE.Group()
    object.add(over, under)

    return {
        object,
        update(zoom: number, width: number, height: number, pixelRatio: number, view: MapView) {
            for (const pass of passes) {
                const {uniforms} = pass.material as THREE.ShaderMaterial
                uniforms.halfViewport.value.set(width / 2, height / 2)
                uniforms.halfWidth.value = halfWidthOf(pixelRatio)
            }

            const paint = flagPaint(zoom, height / pixelRatio, view)
            ;(over.material as THREE.ShaderMaterial).uniforms.ink.value = paint
            over.visible = paint > 0
            under.visible = paint < 1
        },
        dispose() {
            for (const geometry of geometries) geometry.dispose()
            for (const pass of passes) disposeMaterial(pass.material as THREE.Material)
        },
    }
}
