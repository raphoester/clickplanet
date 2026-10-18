import {describe, expect, it} from "vitest"
import {countryIdOf, countryLabel, countryName} from "./countries.ts"

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
