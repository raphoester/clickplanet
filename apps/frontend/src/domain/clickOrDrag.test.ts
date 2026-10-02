import {describe, expect, it} from "vitest"
import {ClickOrDrag, type Pointer} from "./clickOrDrag.ts"

const rules = {mousePx: 6, touchPx: 12}

const mouse = (clientX: number, clientY: number): Pointer =>
    ({pointerId: 1, clientX, clientY, isPrimary: true, pointerType: "mouse"})

const finger = (pointerId: number, clientX: number, clientY: number, isPrimary = true): Pointer =>
    ({pointerId, clientX, clientY, isPrimary, pointerType: "touch"})

describe("ClickOrDrag", () => {
    it("takes a press that stays put for a click", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))

        press.move(mouse(103, 104))

        expect(press.dragged).toBe(false)
    })

    it("takes a press that wanders for a drag of the globe", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))

        press.move(mouse(110, 100))

        expect(press.dragged).toBe(true)
    })

    it("stays a drag when the press comes back to where it began", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))

        press.move(mouse(200, 100))
        press.move(mouse(100, 100))

        expect(press.dragged).toBe(true)
    })

    it("gives a finger more room than a mouse", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))

        press.move(finger(1, 110, 100))
        expect(press.dragged).toBe(false)

        press.move(finger(1, 113, 100))
        expect(press.dragged).toBe(true)
    })

    it("takes a pinch for a drag", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))

        press.begin(finger(2, 400, 400, false))

        expect(press.dragged).toBe(true)
    })

    it("ignores how far another finger goes", () => {
        const press = new ClickOrDrag(rules)
        press.begin(finger(1, 100, 100))

        press.move(finger(2, 400, 400))

        expect(press.dragged).toBe(false)
    })

    it("starts every press as a click", () => {
        const press = new ClickOrDrag(rules)
        press.begin(mouse(100, 100))
        press.move(mouse(300, 100))

        press.begin(mouse(300, 100))

        expect(press.dragged).toBe(false)
    })
})
