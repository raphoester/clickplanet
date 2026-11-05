import {describe, expect, it} from "vitest"
import {fitted} from "./gifs.ts"

const box = {width: 200, height: 200}

describe("fitted", () => {
    it("keeps the shape of a GIF too wide for the box", () => {
        expect(fitted({url: "", width: 220, height: 124}, box)).toEqual({width: 200, height: 113})
    })

    it("keeps the shape of a GIF too tall for the box", () => {
        expect(fitted({url: "", width: 220, height: 400}, box)).toEqual({width: 110, height: 200})
    })

    it("never draws a small GIF bigger than it is", () => {
        expect(fitted({url: "", width: 90, height: 60}, box)).toEqual({width: 90, height: 60})
    })
})
