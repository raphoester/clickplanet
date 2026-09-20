// The countries' outlines, drawn on the tile lattice at a width that does not
// move with the zoom.
//
// The line is not the administrative border. It is the boundary of each
// country's *tiles*: `scripts/generateBorderLines.mjs` walks the cell edges
// between two tiles that belong to different countries and writes them out, so
// the real border is moved by up to half a tile — about 12 km — onto the
// lattice.
//
// That is what makes it readable up close. The tile field is a hex lattice of
// discs covering 76% of the ground once zoomed in, and a cell edge runs exactly
// down the middle of the gap between two of them. So the outline threads
// *between* the tiles: it never cuts one in half, and which side of a border a
// tile is on is never a matter of where the line happened to fall across it.
//
// Zoomed out that gap is gone — the discs are widened to cover the ground so
// the painted flag can reach it — so there the same outline is drawn on the
// other side of the tiles instead, over them. One schedule owns the swap, the
// same `flagPaint` that owns the rest of the handover: the line belongs to
// whichever layer is speaking.
import * as THREE from "three"
import {flagPaint} from "./pointSize.ts"
import {disposeMaterial} from "./scene.ts"

import vertexShader from "./shaders/borderLine/vertex.glsl"
import fragmentShader from "./shaders/borderLine/fragment.glsl"

/** The outline as the blob holds it: corners, grouped into runs. */
export type BorderLineData = {
    /** How many corners each run of the outline has. */
    runs: Uint32Array
    /** Three signed shares of the radius per corner, the runs back to back. */
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

/**
 * How many straight pieces each cell edge is drawn as. One is the bare lattice,
 * corner for corner; above that the run is rounded off, and this is how finely.
 * Four leaves a 15° kink at each join, which a line a pixel and a bit wide and
 * softened at the edges does not show even at the closest zoom.
 */
const SAMPLES = 4

type Run = {at: number, controls: number, closed: boolean}

function runsOf(data: BorderLineData): Run[] {
    const runs: Run[] = []
    let at = 0
    for (const corners of data.runs) {
        // A run that comes back to the corner it started from is a loop, and is
        // rounded right through that corner rather than stopping either side of
        // it. The generator ends a run at every junction, so a corner three
        // countries share is never in the middle of one.
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

/** The control point a curve piece reaches for, wrapped or clamped at the ends. */
function control(run: Run, step: number): number {
    const {at, controls, closed} = run
    if (closed) return at + ((step % controls) + controls) % controls
    return at + Math.min(Math.max(step, 0), controls - 1)
}

/**
 * The outline pulled apart into the straight pieces it is drawn as: one `from`
 * and one `to` per piece, in the blob's own signed shares of the radius.
 *
 * The lattice is a honeycomb, so the bare outline turns 60° at every corner and
 * reads as a staircase up close. What is drawn instead is its **quadratic
 * B-spline**: the curve through the middle of every cell edge, reaching a
 * quarter of the way toward each corner without touching it.
 *
 * That curve is not a compromise on the tile rule — it is the most a curve can
 * be smoothed and still obey it. The narrowest the corridor between two tiles
 * of different countries ever gets is at the middle of the cell edge between
 * them, and the spline goes through that point exactly; at the corners, where
 * there is half as much room again, it uses a fraction of what it has. So the
 * smoothed line clears the tiles by exactly what the staircase cleared them by,
 * and still never crosses one.
 */
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

/**
 * One point of the uniform quadratic B-spline over three control points, at
 * `t` from the middle of the first cell edge to the middle of the second.
 *
 * Both ends fall on a cell edge's midpoint whichever piece works them out, so
 * two pieces that meet land on the same corner to the bit and the line has no
 * seams in it.
 */
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

/**
 * How wide the line is drawn, in drawing-buffer pixels. Thin enough to sit in
 * the gap between two tiles once they are discs — the gap is 24% of the
 * spacing, so it fits from the moment a tile is 6 pixels across, which is
 * inside the handover — and thick enough to survive a globe half a screen wide.
 */
export const WIDTH = 1.3

/** A pixel of softened edge on each side, so the line does not crawl. */
const FEATHER = 1

/**
 * Where each pass sits, as a share of the tiles' own radius of 1.
 *
 * `OVER` is far enough out to clear the tiles and near enough that it is still
 * the same ground. `UNDER` is between the earth's own sphere at 0.999 and the
 * tiles, so the tiles' depth hides the stretch of line they cover and nothing
 * else does.
 */
export const OVER = 1.0005
export const UNDER = 0.9995

/**
 * A dark grey rather than black. Black held its own against a flag and against
 * the sea, but up close, where the line is the only thing between two rows of
 * discs, it read as a bar drawn over the planet instead of a border on it. At a
 * quarter it still carries the whole way out to the globe at half a screen tall
 * — where every border in Europe is on at once — and stops shouting.
 */
const COLOUR = new THREE.Color(0.25, 0.25, 0.25)

export type BorderLines = {
    object: THREE.Object3D
    /** `height` and `width` are the drawing buffer's, in pixels. */
    update(zoom: number, width: number, height: number): void
    dispose(): void
}

export function createBorderLines(data: BorderLineData): BorderLines {
    const {from, to} = outlineSegments(data)
    const pieces = from.length / 3

    const geometry = new THREE.InstancedBufferGeometry()
    // The quad every piece is drawn into: x picks the end, y the side. It is
    // laid out in pixels by the vertex shader, so it carries no position.
    geometry.setAttribute("corner", new THREE.BufferAttribute(
        new Float32Array([-1, -1, -1, 1, 1, -1, 1, 1]), 2,
    ))
    geometry.setIndex([0, 1, 2, 2, 1, 3])
    // Normalized, so the shares of the radius reach the shader as the fractions
    // they are and the whole outline costs half of what floats would.
    geometry.setAttribute("from", new THREE.InstancedBufferAttribute(from, 3, true))
    geometry.setAttribute("to", new THREE.InstancedBufferAttribute(to, 3, true))
    geometry.instanceCount = pieces

    const passes = [OVER, UNDER].map((lift) => {
        const material = new THREE.ShaderMaterial({
            uniforms: {
                halfViewport: {value: new THREE.Vector2(1, 1)},
                halfWidth: {value: WIDTH / 2 + FEATHER},
                lift: {value: lift},
                colour: {value: COLOUR},
                ink: {value: 1},
            },
            vertexShader,
            fragmentShader,
            transparent: true,
            // The quad is laid out in screen space, so which way round it comes
            // out is decided there and not by the geometry: culling either face
            // would throw away the whole outline.
            side: THREE.DoubleSide,
            // The tiles write depth, and that is exactly what hides the under
            // pass where they cover it. Writing any of its own would only let
            // the outline punch holes in whatever is drawn after it.
            depthWrite: false,
        })
        const mesh = new THREE.Mesh(geometry, material)
        // After the tiles, so their depth is already down by the time the under
        // pass is tested against it.
        mesh.renderOrder = 1
        // The quad is built in screen space, so there is nothing here for three
        // to cull against; the outline covers the globe and the globe is the
        // frame.
        mesh.frustumCulled = false
        return mesh
    })
    const [over, under] = passes

    const object = new THREE.Group()
    object.add(over, under)

    return {
        object,
        update(zoom: number, width: number, height: number) {
            for (const pass of passes) {
                (pass.material as THREE.ShaderMaterial).uniforms.halfViewport.value.set(width / 2, height / 2)
            }

            // The over pass belongs to the painted flag and goes out with it;
            // the under pass is always at full strength, and is simply covered
            // by the tiles until they part. Both are the same grey, so the
            // stretch where they overlap — the coasts, which have no tiles on
            // the sea side to hide the under pass — only ever comes out that
            // grey rather than a darker one.
            const paint = flagPaint(zoom, height)
            ;(over.material as THREE.ShaderMaterial).uniforms.ink.value = paint
            over.visible = paint > 0
            under.visible = paint < 1
        },
        dispose() {
            geometry.dispose()
            for (const pass of passes) disposeMaterial(pass.material as THREE.Material)
        },
    }
}
