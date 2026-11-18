import {describe, expect, it} from "vitest"
import {sizedSvg} from "./flagSlots.ts"

describe("sizedSvg", () => {
    it("gives a flag with only a viewBox the viewBox's size", () => {
        expect(sizedSvg(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 900 600"><rect stroke-width="2"/></svg>`, "fr.svg"))
            .toBe(`<?xml version="1.0"?><svg width="900" height="600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 900 600"><rect stroke-width="2"/></svg>`)
    })

    it("takes both from the viewBox when only one is set, and leaves a sized flag as it is", () => {
        expect(sizedSvg(`<svg width='5' viewBox="0,0,30,20"/>`, "x.svg")).toBe(`<svg width="30" height="20" viewBox="0,0,30,20"/>`)
        const sized = `<svg width="30" height="20"><g stroke-width="1"/></svg>`
        expect(sizedSvg(sized, "x.svg")).toBe(sized)
    })

    it("refuses a flag with no size at all", () => {
        expect(() => sizedSvg(`<svg xmlns="http://www.w3.org/2000/svg"/>`, "x.svg")).toThrow("x.svg has no size")
    })
})
