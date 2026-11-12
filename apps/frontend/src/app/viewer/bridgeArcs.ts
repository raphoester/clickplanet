import * as THREE from "three"
import type {Bridge} from "../../domain/bridges.ts"

const BASE = 1.003
// Radians between two dots: about 38 km.
const STEP = 0.006
// How high a bridge's middle rises, as a share of the angle it spans, so a long crossing arches over the sea.
const LIFT = 0.18
const MAX_LIFT = 0.12
const DOT_PX = 3.5
const COLOR = new THREE.Color(0.62, 0.86, 1.0)
const OPACITY = 0.85

// The dots of one arc from one tile to another, ends included, x, y, z each.
export function arcPoints(from: ArrayLike<number>, to: ArrayLike<number>, step = STEP): Float32Array {
    const a = new THREE.Vector3(from[0], from[1], from[2]).normalize()
    const b = new THREE.Vector3(to[0], to[1], to[2]).normalize()
    const angle = a.angleTo(b)
    const dots = Math.max(2, Math.ceil(angle / step))
    const lift = Math.min(MAX_LIFT, LIFT * angle)
    const points = new Float32Array((dots + 1) * 3)
    const at = new THREE.Vector3()
    for (let i = 0; i <= dots; i++) {
        const t = i / dots
        if (angle < 1e-9) at.copy(a)
        else at.copy(a).multiplyScalar(Math.sin((1 - t) * angle)).addScaledVector(b, Math.sin(t * angle))
            .divideScalar(Math.sin(angle))
        at.setLength(BASE + lift * Math.sin(Math.PI * t))
        points.set([at.x, at.y, at.z], i * 3)
    }
    return points
}

export type BridgeArcs = {
    object: THREE.Points
    dispose(): void
}

// positions holds x, y, z per tile, tile id - 1 first.
export function createBridgeArcs(bridges: readonly Bridge[], positions: ArrayLike<number>): BridgeArcs {
    const arcs = bridges.map(([a, b]) => arcPoints(tileAt(positions, a), tileAt(positions, b)))
    const all = new Float32Array(arcs.reduce((sum, arc) => sum + arc.length, 0))
    let offset = 0
    for (const arc of arcs) {
        all.set(arc, offset)
        offset += arc.length
    }

    const geometry = new THREE.BufferGeometry()
    geometry.setAttribute("position", new THREE.BufferAttribute(all, 3))
    const dot = roundDot()
    const material = new THREE.PointsMaterial({
        color: COLOR,
        size: DOT_PX,
        map: dot,
        transparent: true,
        opacity: OPACITY,
        depthWrite: false,
    })
    const object = new THREE.Points(geometry, material)

    return {
        object,
        dispose() {
            geometry.dispose()
            material.dispose()
            dot.dispose()
        },
    }
}

function tileAt(positions: ArrayLike<number>, tile: number): number[] {
    const i = (tile - 1) * 3
    return [positions[i], positions[i + 1], positions[i + 2]]
}

function roundDot(): THREE.DataTexture {
    const size = 16
    const data = new Uint8Array(size * size * 4)
    for (let y = 0; y < size; y++) {
        for (let x = 0; x < size; x++) {
            const d = Math.hypot(x + 0.5 - size / 2, y + 0.5 - size / 2) / (size / 2)
            const i = (y * size + x) * 4
            data.set([255, 255, 255, Math.round(255 * Math.min(1, Math.max(0, (1 - d) * 4)))], i)
        }
    }
    const texture = new THREE.DataTexture(data, size, size, THREE.RGBAFormat)
    texture.magFilter = THREE.LinearFilter
    texture.minFilter = THREE.LinearFilter
    texture.needsUpdate = true
    return texture
}
