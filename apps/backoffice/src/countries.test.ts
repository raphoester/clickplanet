import {describe, expect, it} from "vitest"
import {countryIdOf, countryLabel, countryName, flagOf} from "./countries.ts"

describe("countryName", () => {
    it("names an ISO code, however it was typed", () => {
        expect(countryName("fr")).toBe("France")
        expect(countryName(" DZ ")).toBe("Algeria")
    })

    it("names the game's own ids, which no standard names", () => {
        expect(countryName("gb-sct")).toBe("Scotland")
        expect(countryName("xb")).toBe("Brittany")
    })

    it("names nothing it does not know", () => {
        expect(countryName("france")).toBeUndefined()
        expect(countryName("")).toBeUndefined()
    })
})

describe("countryLabel", () => {
    it("is the name, or the id when there is none", () => {
        expect(countryLabel("dz")).toBe("Algeria")
        expect(countryLabel("zz-top")).toBe("ZZ-TOP")
    })
})

describe("countryIdOf", () => {
    it("is the id the server takes", () => {
        expect(countryIdOf(" GB-ENG ")).toBe("gb-eng")
    })
})

describe("flagOf", () => {
    it("is the emoji of an ISO code", () => {
        expect(flagOf("fr")).toBe("\u{1F1EB}\u{1F1F7}")
        expect(flagOf("ps")).toBe("\u{1F1F5}\u{1F1F8}")
    })

    it("is the tag sequence of the nations Unicode draws a flag for", () => {
        expect(flagOf("gb-sct")).toBe("\u{1F3F4}\u{E0067}\u{E0062}\u{E0073}\u{E0063}\u{E0074}\u{E007F}")
    })

    it("is nothing for an id with no flag, rather than two letters in boxes", () => {
        expect(flagOf("xb")).toBe("")
        expect(flagOf("zz-top")).toBe("")
        expect(flagOf("")).toBe("")
    })
})
