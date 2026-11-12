import {describe, expect, it} from "vitest"
import {championNames, roundOverLine, winsLine} from "./seasonLines.ts"

describe("winsLine", () => {
    it("names the champion and its season", () => {
        expect(winsLine({season: 0, countryCodes: ["de"]})).toBe("Germany wins Season 0")
    })

    it("names champions together", () => {
        expect(winsLine({season: 2, countryCodes: ["de", "fr", "es"]})).toBe("Germany, France, and Spain win Season 2")
        expect(championNames({season: 2, countryCodes: ["de", "fr"]})).toBe("Germany and France")
    })
})

describe("roundOverLine", () => {
    it("names the day, the battle, or the Final Battle", () => {
        expect(roundOverLine({number: 5, finale: false, own: false})).toBe("Day 5 is over!")
        expect(roundOverLine({number: 23, finale: true, own: false})).toBe("Day 23 is over!")
        expect(roundOverLine({number: 1, finale: false, own: true})).toBe("The battle is over!")
        expect(roundOverLine({number: 2, finale: true, own: true})).toBe("The Final Battle is over!")
    })
})
