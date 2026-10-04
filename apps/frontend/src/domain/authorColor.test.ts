import {describe, expect, it} from "vitest"
import {NameColor} from "../backends/player.ts"
import {hueOf, NAME_COLORS} from "./authorColor.ts"

describe("hueOf", () => {
    it("is the color's hue", () => {
        expect(hueOf(NameColor.TEAL)).toBe(165)
    })

    it("is nothing when no color was chosen", () => {
        expect(hueOf(NameColor.UNSPECIFIED)).toBeUndefined()
        expect(hueOf(undefined)).toBeUndefined()
    })

    it("is nothing for a color this build does not know", () => {
        expect(hueOf(99 as NameColor)).toBeUndefined()
    })
})

describe("NAME_COLORS", () => {
    it("offers every color the proto names, each once, and each a hue of its own", () => {
        const named = Object.values(NameColor).filter((value): value is NameColor =>
            typeof value === "number" && value !== NameColor.UNSPECIFIED)

        expect(NAME_COLORS.map((choice) => choice.color).sort((a, b) => a - b)).toEqual(named)
        expect(new Set(NAME_COLORS.map((choice) => choice.hue)).size).toBe(NAME_COLORS.length)
    })
})
