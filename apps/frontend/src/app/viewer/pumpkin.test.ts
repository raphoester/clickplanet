import * as THREE from "three"
import {describe, expect, it} from "vitest"
import {flickerAt, pumpkinGeometry, tumbleAt} from "./pumpkin.ts"

function vertexNearest(geometry: THREE.BufferGeometry, u: number, v: number): THREE.Vector3 {
    const uv = geometry.attributes.uv
    let best = 0
    for (let i = 1; i < uv.count; i++) {
        if (Math.hypot(uv.getX(i) - u, uv.getY(i) - v) < Math.hypot(uv.getX(best) - u, uv.getY(best) - v)) best = i
    }
    return new THREE.Vector3().fromBufferAttribute(geometry.attributes.position, best)
}

describe("pumpkinGeometry", () => {
    it("carves its face where it looks down +z, and its question marks on its sides and back", () => {
        const geometry = pumpkinGeometry()
        expect(vertexNearest(geometry, 0.5, 0.5).normalize().z).toBeCloseTo(1, 1)
        expect(Math.abs(vertexNearest(geometry, 0.25, 0.5).normalize().x)).toBeCloseTo(1, 1)
        expect(Math.abs(vertexNearest(geometry, 0.75, 0.5).normalize().x)).toBeCloseTo(1, 1)
        expect(vertexNearest(geometry, 0.01, 0.5).normalize().z).toBeCloseTo(-1, 1)
    })

    it("is wider than tall, with grooves between its lobes", () => {
        const geometry = pumpkinGeometry()
        geometry.computeBoundingBox()
        const size = geometry.boundingBox!.getSize(new THREE.Vector3())
        expect(size.x).toBeGreaterThan(size.y * 1.15)

        const groove = vertexNearest(geometry, 0.5, 0.5).length()
        const lobe = vertexNearest(geometry, 0.5 + 1 / 16, 0.5).length()
        expect(lobe - groove).toBeGreaterThan(0.04)
    })
})

describe("flickerAt", () => {
    it("wavers a little round full light, never out", () => {
        for (let frame = 0; frame < 600; frame++) {
            const flicker = flickerAt(frame / 60)
            expect(flicker).toBeGreaterThanOrEqual(0.76)
            expect(flicker).toBeLessThanOrEqual(1)
        }
    })
})

describe("tumbleAt", () => {
    it("turns and rocks on all three axes, and never points its stem down", () => {
        const up = new THREE.Vector3(0, 1, 0)
        const front = new THREE.Vector3(0, 0, 1)
        let lowest = 1
        let leanX = 0
        let leanZ = 0
        let faced = 0
        for (let frame = 0; frame < 60 * 60; frame++) {
            const turn = tumbleAt(frame / 60)
            const stem = up.clone().applyQuaternion(turn)
            lowest = Math.min(lowest, stem.y)
            leanX = Math.max(leanX, Math.abs(stem.x))
            leanZ = Math.max(leanZ, Math.abs(stem.z))
            if (front.clone().applyQuaternion(turn).z > 0.9) faced++
        }
        expect(lowest).toBeGreaterThan(0.4)
        expect(leanX).toBeGreaterThan(0.3)
        expect(leanZ).toBeGreaterThan(0.3)
        expect(faced).toBeGreaterThan(60)
    })
})
