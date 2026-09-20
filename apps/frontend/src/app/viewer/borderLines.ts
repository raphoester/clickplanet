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
 * The runs pulled apart into the edges they are made of: one `from` and one
 * `to` per edge. A run of n corners is n-1 edges, which is what keeps the file
 * to one corner per edge instead of two.
 *
 * The coordinates stay as the blob's own fractions of the radius — the shader
 * normalizes what it reads, so their scale never matters.
 */
export function borderSegments(data: BorderLineData): {from: Float32Array, to: Float32Array} {
    let edges = 0
    for (const length of data.runs) edges += Math.max(0, length - 1)

    const from = new Float32Array(edges * 3)
    const to = new Float32Array(edges * 3)

    let corner = 0
    let edge = 0
    for (const length of data.runs) {
        for (let step = 0; step + 1 < length; step++) {
            const head = (corner + step) * 3
            const tail = head + 3
            for (let axis = 0; axis < 3; axis++) {
                from[edge * 3 + axis] = data.corners[head + axis]
                to[edge * 3 + axis] = data.corners[tail + axis]
            }
            edge++
        }
        corner += length
    }

    return {from, to}
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

/** Not quite black, as the keyline around a painted flag is not. */
const COLOUR = new THREE.Color(0.03, 0.03, 0.03)

export type BorderLines = {
    object: THREE.Object3D
    /** `height` and `width` are the drawing buffer's, in pixels. */
    update(zoom: number, width: number, height: number): void
    dispose(): void
}

export function createBorderLines(data: BorderLineData): BorderLines {
    const {from, to} = borderSegments(data)
    const edges = from.length / 3

    const geometry = new THREE.InstancedBufferGeometry()
    // The quad every edge is drawn into: x picks the end, y the side. It is
    // laid out in pixels by the vertex shader, so it carries no position.
    geometry.setAttribute("corner", new THREE.BufferAttribute(
        new Float32Array([-1, -1, -1, 1, 1, -1, 1, 1]), 2,
    ))
    geometry.setIndex([0, 1, 2, 2, 1, 3])
    geometry.setAttribute("from", new THREE.InstancedBufferAttribute(from, 3))
    geometry.setAttribute("to", new THREE.InstancedBufferAttribute(to, 3))
    geometry.instanceCount = edges

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
            // by the tiles until they part. Both are the same black, so the
            // stretch where they overlap — the coasts, which have no tiles on
            // the sea side to hide the under pass — only ever comes out black.
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
