import {describe, expect, it} from "vitest"
import {draggedSize, parseStoredSize} from "./chatSize.ts"

describe("parseStoredSize", () => {
    it("reads back what was stored", () => {
        const raw = JSON.stringify({width: 520, height: 640})

        expect(parseStoredSize(raw)).toEqual({width: 520, height: 640})
    })

    it("gives up on anything it cannot read", () => {
        expect(parseStoredSize(null)).toBeUndefined()
        expect(parseStoredSize("")).toBeUndefined()
        expect(parseStoredSize("not json")).toBeUndefined()
        expect(parseStoredSize("42")).toBeUndefined()
        expect(parseStoredSize(JSON.stringify({width: 520}))).toBeUndefined()
        expect(parseStoredSize(JSON.stringify({width: "520", height: 640}))).toBeUndefined()
        expect(parseStoredSize(JSON.stringify({width: 0, height: 640}))).toBeUndefined()
        expect(parseStoredSize(JSON.stringify({width: 520, height: -1}))).toBeUndefined()
    })
})

describe("draggedSize", () => {
    const start = {width: 360, height: 460}

    it("grows up and to the left, away from the corner the panel is pinned to", () => {
        expect(draggedSize("corner", start, -100, -50)).toEqual({width: 460, height: 510})
    })

    it("moves only the height from the top edge", () => {
        expect(draggedSize("top", start, -100, -50)).toEqual({width: 360, height: 510})
    })

    it("moves only the width from the left edge", () => {
        expect(draggedSize("left", start, -100, -50)).toEqual({width: 460, height: 460})
    })

    it("shrinks when dragged back toward the corner", () => {
        expect(draggedSize("corner", start, 40, 60)).toEqual({width: 320, height: 400})
    })
})
