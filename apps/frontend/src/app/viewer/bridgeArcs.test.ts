import {describe, expect, it} from "vitest"
import {arcPoints, createBridgeArcs} from "./bridgeArcs.ts"

const radius = (points: Float32Array, i: number) => Math.hypot(points[i * 3], points[i * 3 + 1], points[i * 3 + 2])

describe("arcPoints", () => {
    it("runs from one end to the other, just over the ground at both", () => {
        const points = arcPoints([1, 0, 0], [0, 0, 1])
        const last = points.length / 3 - 1

        expect(points[0] / radius(points, 0)).toBeCloseTo(1)
        expect(points[last * 3 + 2] / radius(points, last)).toBeCloseTo(1)
        expect(radius(points, 0)).toBeCloseTo(1.003)
        expect(radius(points, last)).toBeCloseTo(1.003)
    })

    it("rises highest in the middle of a long crossing", () => {
        const points = arcPoints([1, 0, 0], [0, 0, 1])
        const middle = (points.length / 3 - 1) / 2

        expect(radius(points, middle)).toBeGreaterThan(1.1)
    })

    it("puts more dots on a longer crossing, and three on a short one", () => {
        expect(arcPoints([1, 0, 0], [1, 0, 0.001]).length / 3).toBe(3)
        expect(arcPoints([1, 0, 0], [0, 0, 1]).length).toBeGreaterThan(arcPoints([1, 0, 0], [1, 0, 0.5]).length)
    })
})

describe("createBridgeArcs", () => {
    it("draws every bridge's dots in one object", () => {
        const positions = [1, 0, 0, 0, 0, 1, 0, 1, 0]
        const arcs = createBridgeArcs([[1, 2], [2, 3]], positions)

        const expected = arcPoints([1, 0, 0], [0, 0, 1]).length + arcPoints([0, 0, 1], [0, 1, 0]).length
        expect(arcs.object.geometry.getAttribute("position").array.length).toBe(expected)
        arcs.dispose()
    })
})
