import {describe, expect, it} from "vitest"
import {QuizQuestion, stillOpen, timeLeft} from "./quiz.ts"

const question = (deadline: number, window = 5000): QuizQuestion =>
    ({text: "?", choices: ["a", "b", "c"], deadline, window})

describe("timeLeft", () => {
    it("is the share of the window still to run", () => {
        expect(timeLeft(question(5000), 0)).toBe(1)
        expect(timeLeft(question(5000), 2500)).toBe(0.5)
        expect(timeLeft(question(5000), 5000)).toBe(0)
    })

    it("is empty rather than negative once the deadline has passed", () => {
        // The bar is drawn from this, and a negative scale is a bar drawn backwards.
        expect(timeLeft(question(5000), 9000)).toBe(0)
    })

    it("is full rather than over one when the clock ran backwards", () => {
        expect(timeLeft(question(5000), -1000)).toBe(1)
    })

    it("is empty for a window of nothing, rather than dividing by zero", () => {
        // A server too old to say how long it gave answers zero seconds.
        expect(timeLeft(question(5000, 0), 0)).toBe(0)
    })
})

describe("stillOpen", () => {
    it("closes on the deadline, not after it", () => {
        expect(stillOpen(question(5000), 4999)).toBe(true)
        expect(stillOpen(question(5000), 5000)).toBe(false)
    })
})
