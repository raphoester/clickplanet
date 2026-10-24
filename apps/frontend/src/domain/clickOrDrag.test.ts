import {describe, expect, it} from "vitest"
import {ClickOrDrag, type Pointer, type Spot} from "./clickOrDrag.ts"

const rules = {mousePx: 6, touchPx: 12}

const mouse = (clientX: number, clientY: number): Pointer =>
    ({pointerId: 1, clientX, clientY, isPrimary: true, pointerType: "mouse"})

const finger = (pointerId: number, clientX: number, clientY: number, isPrimary = true): Pointer =>
    ({pointerId, clientX, clientY, isPrimary, pointerType: "touch"})

const spot = (clientX: number, clientY: number): Spot => ({clientX, clientY})

describe("ClickOrDrag", () => {
    it("takes a press that stays put for a click", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))

        press.move(mouse(103, 104))

        expect(press.spotOf(spot(100, 100))).toEqual(spot(100, 100))
    })

    it("takes a press that wanders for a drag of the globe", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))

        press.move(mouse(110, 100))

        expect(press.spotOf(spot(100, 100))).toBeUndefined()
    })

    it("stays a drag when the press comes back to where it began", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))

        press.move(mouse(200, 100))
        press.move(mouse(100, 100))

        expect(press.spotOf(spot(100, 100))).toBeUndefined()
    })

    it("gives a finger more room than a mouse", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))

        press.move(finger(1, 110, 100))
        expect(press.spotOf(spot(100, 100))).toEqual(spot(100, 100))

        press.move(finger(1, 113, 100))
        expect(press.spotOf(spot(100, 100))).toBeUndefined()
    })

    it("takes a pinch for a drag", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))

        press.begin(finger(2, 400, 400, false))

        expect(press.spotOf(spot(100, 100))).toBeUndefined()
    })

    it("ignores how far another finger goes", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))

        press.move(finger(2, 400, 400))

        expect(press.spotOf(spot(100, 100))).toEqual(spot(100, 100))
    })

    it("starts every press as a click", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))
        press.move(mouse(300, 100))

        press.begin(mouse(300, 100))

        expect(press.spotOf(spot(100, 100))).toEqual(spot(100, 100))
    })

    it("puts the click where the press lifted", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))

        press.end(finger(1, 104, 102))

        expect(press.spotOf(spot(40, 40))).toEqual(spot(104, 102))
    })

    it("puts a quick second tap on its own tile, not on the first one", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))
        press.end(finger(1, 100, 100))
        expect(press.spotOf(spot(100, 100))).toEqual(spot(100, 100))

        press.begin(finger(2, 130, 100))
        press.end(finger(2, 130, 100))

        expect(press.spotOf(spot(100, 100))).toEqual(spot(130, 100))
    })

    it("ignores where another finger lifts", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))

        press.end(finger(2, 400, 400))

        expect(press.spotOf(spot(100, 100))).toEqual(spot(100, 100))
    })

    it("forgets where the last press lifted", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))
        press.end(mouse(100, 100))

        press.begin(mouse(300, 100))

        expect(press.spotOf(spot(300, 100))).toEqual(spot(300, 100))
    })
})
