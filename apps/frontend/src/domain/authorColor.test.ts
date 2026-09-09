import {describe, expect, it} from "vitest"
import {authorHue} from "./authorColor.ts"

describe("authorHue", () => {
    it("is the same every time for the same author", () => {
        expect(authorHue("Ana", "4f2ca1")).toBe(authorHue("Ana", "4f2ca1"))
    })

    it("is a hue", () => {
        const names = ["", "Ana", "Bo", "Théo", "🐧", "x".repeat(24)]
        for (const name of names) {
            const hue = authorHue(name, "4f2ca1")
            expect(hue).toBeGreaterThanOrEqual(0)
            expect(hue).toBeLessThan(360)
            expect(Number.isInteger(hue)).toBe(true)
        }
    })

    it("separates two people typing the same name", () => {
        expect(authorHue("Ana", "4f2ca1")).not.toBe(authorHue("Ana", "c0ffee"))
    })

    it("separates two names behind the same tag", () => {
        expect(authorHue("Ana", "4f2ca1")).not.toBe(authorHue("Bo", "4f2ca1"))
    })

    it("spreads a handful of authors over the wheel rather than bunching them", () => {
        const hues = ["Ana", "Bo", "Cy", "Dee", "Eli", "Fay", "Gus", "Hal"]
            .map(name => authorHue(name, "4f2ca1"))

        expect(new Set(hues).size).toBe(hues.length)
    })
})
