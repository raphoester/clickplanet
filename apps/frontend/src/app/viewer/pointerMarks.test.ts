// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {burstPoints} from "./blastMark.ts"
import {createBonusPointer} from "./bonusPointer.ts"
import {QUESTION_MARK} from "./questionMark.ts"

describe("the pointer's badge", () => {
    it("carries the box's drawn question mark, not a typed one", () => {
        const container = document.createElement("div")
        const pointer = createBonusPointer(container)

        const badge = container.querySelector(".bonus-pointer-badge")!
        expect(badge.querySelector("svg path")?.getAttribute("d")).toBe(QUESTION_MARK.hook)
        expect(badge.textContent).toBe("")

        pointer.dispose()
    })

    it("carries a drawn burst for a blast, not an emoji", () => {
        const container = document.createElement("div")
        const pointer = createBonusPointer(container, "blast")

        const badge = container.querySelector(".bonus-pointer--blast .bonus-pointer-badge")!
        expect(badge.querySelectorAll("svg polygon")).toHaveLength(2)
        expect(badge.textContent).toBe("")

        pointer.dispose()
    })
})

describe("the question mark", () => {
    it("stays inside its square, stroke and dot included", () => {
        const {size, strokeWidth, dot} = QUESTION_MARK
        const numbers = QUESTION_MARK.hook.match(/-?\d+(\.\d+)?/g)!.map(Number)

        for (const value of numbers) {
            expect(value - strokeWidth / 2).toBeGreaterThanOrEqual(0)
            expect(value + strokeWidth / 2).toBeLessThanOrEqual(size)
        }
        expect(dot.cy + dot.r).toBeLessThanOrEqual(size)
    })
})

describe("burstPoints", () => {
    it("alternates a spike and a notch, all the way round", () => {
        const points = burstPoints(10, 44, 25).split(" ").map((pair) => {
            const [x, y] = pair.split(",").map(Number)
            return new THREE.Vector2(x - 50, y - 50).length()
        })

        expect(points).toHaveLength(20)
        points.forEach((radius, i) => expect(radius).toBeCloseTo(i % 2 === 0 ? 44 : 25, 1))
    })

    it("starts with a spike straight up", () => {
        const [x, y] = burstPoints(8, 40, 20).split(" ")[0].split(",").map(Number)

        expect(x).toBeCloseTo(50, 5)
        expect(y).toBeCloseTo(10, 5)
    })
})
