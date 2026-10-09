import {describe, expect, it} from "vitest"
import {tollRows} from "./toll.ts"

const STEPS = [
    {share: 0.1, slowdown: 1.5},
    {share: 0.2, slowdown: 2.5},
    {share: 0.3, slowdown: 4},
]

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
