import {describe, expect, it} from "vitest"
import {cardPlacement} from "./cardPlacement.ts"

const CARD = {width: 260, height: 200}
const DESKTOP = {width: 1280, height: 800}
const PHONE = {width: 375, height: 812}

describe("cardPlacement", () => {
    it("puts the card below and right of the mouse", () => {
        expect(cardPlacement({x: 400, y: 300}, CARD, DESKTOP, false)).toEqual({x: 416, y: 316})
    })

    it("turns to the left of the mouse, or above it, where the card would leave the screen", () => {
        expect(cardPlacement({x: 1200, y: 300}, CARD, DESKTOP, false)).toEqual({x: 924, y: 316})
        expect(cardPlacement({x: 400, y: 700}, CARD, DESKTOP, false)).toEqual({x: 416, y: 484})
    })

    it("puts the card above a finger, centred on it, where the finger does not cover it", () => {
        expect(cardPlacement({x: 180, y: 500}, CARD, PHONE, true)).toEqual({x: 50, y: 252})
    })

    it("puts it below a finger near the top, and keeps it on screen at the sides", () => {
        expect(cardPlacement({x: 20, y: 120}, CARD, PHONE, true)).toEqual({x: 12, y: 168})
        expect(cardPlacement({x: 370, y: 500}, CARD, PHONE, true)).toEqual({x: 103, y: 252})
    })
})
