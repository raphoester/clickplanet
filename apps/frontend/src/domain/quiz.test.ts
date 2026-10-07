import {describe, expect, it} from "vitest"
import {QuizQuestion, secondsLeft, stillOpen, timeLeft, untilNextSecond} from "./quiz.ts"

const question = (deadline: number, window = 5000): QuizQuestion =>
    ({text: "?", choices: ["a", "b", "c"], deadline, window})

describe("timeLeft", () => {
    it("is the share of the window still to run", () => {
        expect(timeLeft(question(5000), 0)).toBe(1)
        expect(timeLeft(question(5000), 2500)).toBe(0.5)
        expect(timeLeft(question(5000), 5000)).toBe(0)
    })

    it("is empty rather than negative once the deadline has passed", () => {
        expect(timeLeft(question(5000), 9000)).toBe(0)
    })

    it("is full rather than over one when the clock ran backwards", () => {
        expect(timeLeft(question(5000), -1000)).toBe(1)
    })

    it("is empty for a window of nothing, rather than dividing by zero", () => {
        expect(timeLeft(question(5000, 0), 0)).toBe(0)
    })
})

describe("secondsLeft", () => {
    it("counts a second that has started as a whole one", () => {
        expect(secondsLeft(question(5000), 0)).toBe(5)
        expect(secondsLeft(question(5000), 1)).toBe(5)
        expect(secondsLeft(question(5000), 1000)).toBe(4)
        expect(secondsLeft(question(5000), 4999)).toBe(1)
    })

    it("is nothing on the deadline and after it", () => {
        expect(secondsLeft(question(5000), 5000)).toBe(0)
        expect(secondsLeft(question(5000), 9000)).toBe(0)
    })
})

describe("untilNextSecond", () => {
    it("is the wait until the count goes down by one", () => {
        expect(untilNextSecond(question(5000), 0)).toBe(1000)
        expect(untilNextSecond(question(5000), 300)).toBe(700)
        expect(untilNextSecond(question(5000), 4999)).toBe(1)
    })

    it("is forever once the count is at nothing", () => {
        expect(untilNextSecond(question(5000), 5000)).toBe(Infinity)
    })
})

describe("stillOpen", () => {
    it("closes on the deadline, not after it", () => {
        expect(stillOpen(question(5000), 4999)).toBe(true)
        expect(stillOpen(question(5000), 5000)).toBe(false)
    })
})
