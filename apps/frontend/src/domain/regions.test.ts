import {describe, expect, it} from "vitest"
import {regionOf} from "./regions.ts"

describe("the region of a country", () => {
    it("is its continent", () => {
        expect(regionOf("fr")).toBe("Europe")
        expect(regionOf("dz")).toBe("Africa")
        expect(regionOf("br")).toBe("South America")
    })

    it("is the part of Asia people name, Asia being too big to be one place", () => {
        expect(regionOf("sa")).toBe("Middle East")
        expect(regionOf("ir")).toBe("Middle East")
        expect(regionOf("tr")).toBe("Middle East")
        expect(regionOf("in")).toBe("South Asia")
        expect(regionOf("cn")).toBe("East Asia")
        expect(regionOf("id")).toBe("Southeast Asia")
        expect(regionOf("kz")).toBe("Central Asia")
    })
})
