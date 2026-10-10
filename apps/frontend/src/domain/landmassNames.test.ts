import {describe, expect, it} from "vitest"
import {LANDMASS_NAMES_BORDERS, landmassName} from "./landmassNames.ts"
import {BORDERS_URL} from "../app/viewer/bordersAsset.ts"

describe("landmassName", () => {
    it("was generated for the borders blob this build fetches", () => {
        expect(`/static/${LANDMASS_NAMES_BORDERS}`).toBe(BORDERS_URL)
    })

    it("names a landmass by its own name, and a country's main one by the country", () => {
        expect(landmassName(257, "fr")).toBe("Corsica")
        expect(landmassName(228, "fr")).toBe("France")
    })
})
