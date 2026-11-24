import {describe, expect, it} from "vitest"
import {countryIdOf, countryLabel, countryName, flagOf} from "./countries.ts"

describe("countryName", () => {
    it("names an ISO code, however it was typed", () => {
        expect(countryName("fr")).toBe("France")
        expect(countryName(" DZ ")).toBe("Algeria")
    })

    it("names Kosovo, which ISO gives no code", () => {
        expect(countryName("xk")).toBe("Kosovo")
    })

    it("names nothing it does not know", () => {
        expect(countryName("france")).toBeUndefined()
        expect(countryName("")).toBeUndefined()
    })

    it("takes no name Intl gives a private-use code as a test, as it does Brittany's", () => {
        expect(countryName("xb")).toBeUndefined()
        expect(countryName("xa")).toBeUndefined()
    })

    it("names a flag that left the build only from the server's list, as every sold flag", () => {
        expect(countryName("gb-sct")).toBeUndefined()
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

    it("is nothing for an id with no flag, rather than two letters in boxes", () => {
        expect(flagOf("xb")).toBe("")
        expect(flagOf("gb-sct")).toBe("")
        expect(flagOf("zz-top")).toBe("")
        expect(flagOf("")).toBe("")
    })
})
