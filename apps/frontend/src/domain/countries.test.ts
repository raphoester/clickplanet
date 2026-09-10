import {describe, expect, it} from "vitest"
import {Countries, nameWithoutFlag} from "./countries.ts"

describe("nameWithoutFlag", () => {
    it("takes the flag off a name", () => {
        expect(nameWithoutFlag(Countries.get("fr")!)).toBe("France")
    })

    it("takes off a flag written as a tag sequence", () => {
        expect(nameWithoutFlag(Countries.get("gb-eng")!)).toBe("England")
    })

    it("takes off a flag that is not a country flag", () => {
        expect(nameWithoutFlag(Countries.get("xb")!)).toBe("Brittany")
    })

    it("leaves no emoji in any country's name", () => {
        const emojis = Array.from(Countries.values())
            .map(nameWithoutFlag)
            .filter(name => /\p{RI}|\p{Extended_Pictographic}/u.test(name))
        expect(emojis).toEqual([])
    })

    it("leaves every country with a name", () => {
        const empty = Array.from(Countries.values())
            .filter(country => nameWithoutFlag(country).trim() === "")
        expect(empty).toEqual([])
    })
})
