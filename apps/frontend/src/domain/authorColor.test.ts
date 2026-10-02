import {describe, expect, it} from "vitest"
import {NameColor} from "../backends/player.ts"
import {authorHue, NAME_COLORS} from "./authorColor.ts"

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

describe("authorHue with a chosen color", () => {
    it("is the color's hue, whatever the name", () => {
        expect(authorHue("Ana", NameColor.TEAL)).toBe(165)
        expect(authorHue("Bo", NameColor.TEAL)).toBe(165)
    })

    it("is the name's hue when no color was chosen", () => {
        expect(authorHue("Ana", NameColor.UNSPECIFIED)).toBe(authorHue("Ana"))
    })

    it("is the name's hue for a color this build does not know", () => {
        expect(authorHue("Ana", 99 as NameColor)).toBe(authorHue("Ana"))
    })

    it("offers every color the proto names, each once, and each a hue of its own", () => {
        const named = Object.values(NameColor).filter((value): value is NameColor =>
            typeof value === "number" && value !== NameColor.UNSPECIFIED)

        expect(NAME_COLORS.map((choice) => choice.color).sort((a, b) => a - b)).toEqual(named)
        expect(new Set(NAME_COLORS.map((choice) => choice.hue)).size).toBe(NAME_COLORS.length)
    })
})
