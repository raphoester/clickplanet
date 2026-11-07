import {describe, expect, it} from "vitest"
import {Fingers} from "./fingers.ts"

describe("Fingers", () => {
    it("finds no stray while the browser counts as many fingers as were pressed", () => {
        const fingers = new Fingers()
        fingers.press(7)
        fingers.press(8)

        expect(fingers.strayBeyond(2)).toEqual([])
        expect(fingers.strayBeyond(0)).toEqual([7, 8])
    })

    it("finds no stray while the browser counts more, such as a finger on a panel", () => {
        const fingers = new Fingers()
        fingers.press(7)

        expect(fingers.strayBeyond(3)).toEqual([])
    })

    it("gives back the finger iOS took when the next one is the only one on the screen", () => {
        const fingers = new Fingers()
        fingers.press(7)
        fingers.press(8)

        expect(fingers.strayBeyond(1)).toEqual([7])
        expect(fingers.strayBeyond(0)).toEqual([8])
    })

    it("gives every finger back once none is on the screen", () => {
        const fingers = new Fingers()
        fingers.press(7)
        fingers.press(8)

        expect(fingers.strayBeyond(0)).toEqual([7, 8])
    })

    it("gives a stray back once, not again on the next count", () => {
        const fingers = new Fingers()
        fingers.press(7)
        fingers.press(8)
        fingers.strayBeyond(1)

        expect(fingers.strayBeyond(1)).toEqual([])
    })

    it("forgets a finger lifted, and counts a finger pressed twice once", () => {
        const fingers = new Fingers()
        fingers.press(7)
        fingers.press(7)
        fingers.press(8)
        fingers.lift(7)

        expect(fingers.strayBeyond(0)).toEqual([8])
    })
})
