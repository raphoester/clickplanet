import {describe, expect, it} from "vitest"
import {Countries} from "./countries.ts"

// The flag used to be an emoji in front of every name, and every place that drew
// a name had to take it back off. The flags come from the atlas now, so the
// names are just names — and this is what keeps them that way.
const EMOJI = /\p{RI}|\p{Extended_Pictographic}/u

describe("Countries", () => {
    it("names a country in plain words", () => {
        expect(Countries.get("fr")!.name).toBe("France")
        expect(Countries.get("gb-eng")!.name).toBe("England")
        expect(Countries.get("xb")!.name).toBe("Brittany")
    })

    it("carries no emoji in any name", () => {
        const decorated = Array.from(Countries.values())
            .filter(country => EMOJI.test(country.name))
            .map(country => `${country.code}: ${country.name}`)
        expect(decorated).toEqual([])
    })

    it("gives every country a name of its own, with no space around it", () => {
        const wrong = Array.from(Countries.values())
            .filter(country => country.name === "" || country.name !== country.name.trim())
            .map(country => country.code)
        expect(wrong).toEqual([])
    })

    it("keys every country by its own code", () => {
        const mismatched = Array.from(Countries.entries())
            .filter(([code, country]) => code !== country.code)
            .map(([code]) => code)
        expect(mismatched).toEqual([])
    })
})
