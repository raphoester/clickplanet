import {describe, expect, it} from "vitest"
import {authorHue} from "./authorColor.ts"

describe("authorHue", () => {
    it("is the same every time for the same author", () => {
        expect(authorHue("Ana")).toBe(authorHue("Ana"))
    })

    it("is a hue", () => {
        const names = ["", "Ana", "Bo", "Théo", "🐧", "x".repeat(24)]
        for (const name of names) {
            const hue = authorHue(name)
            expect(hue).toBeGreaterThanOrEqual(0)
            expect(hue).toBeLessThan(360)
            expect(Number.isInteger(hue)).toBe(true)
        }
    })

    it("separates two names", () => {
        expect(authorHue("Ana")).not.toBe(authorHue("Bo"))
        expect(authorHue("guest_91aa3d")).not.toBe(authorHue("guest_c0ffee"))
    })

    it("spreads a handful of authors over the wheel rather than bunching them", () => {
        const hues = ["Ana", "Bo", "Cy", "Dee", "Eli", "Fay", "Gus", "Hal"]
            .map(name => authorHue(name))

        expect(new Set(hues).size).toBe(hues.length)
    })
})
