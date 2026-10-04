import {describe, expect, it} from "vitest"
import {slowdownAt, tollRows} from "./toll.ts"

const STEPS = [
    {share: 0.1, slowdown: 1.5},
    {share: 0.2, slowdown: 2.5},
    {share: 0.3, slowdown: 4},
]

describe("slowdownAt", () => {
    it("is the plain rate under the first step", () => {
        expect(slowdownAt(STEPS, 0.099)).toBe(1)
    })

    it("takes a step from its share, as the server does", () => {
        expect(slowdownAt(STEPS, 0.1)).toBe(1.5)
        expect(slowdownAt(STEPS, 0.25)).toBe(2.5)
    })

    it("stays on the last step past it", () => {
        expect(slowdownAt(STEPS, 0.98)).toBe(4)
    })

    it("is the plain rate with no steps", () => {
        expect(slowdownAt([], 0.98)).toBe(1)
    })
})

describe("tollRows", () => {
    it("starts with the plain rate and marks the step the share is on", () => {
        expect(tollRows(STEPS, 0.98)).toEqual([
            {share: 0, slowdown: 1, here: false},
            {share: 0.1, slowdown: 1.5, here: false},
            {share: 0.2, slowdown: 2.5, here: false},
            {share: 0.3, slowdown: 4, here: true},
        ])
    })

    it("marks the plain rate for a small country", () => {
        expect(tollRows(STEPS, 0.001).map((row) => row.here)).toEqual([true, false, false, false])
    })
})
