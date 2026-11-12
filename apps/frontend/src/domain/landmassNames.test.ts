import {describe, expect, it} from "vitest"
import {LANDMASS_NAMES_BORDERS} from "./landmassNames.ts"
import {BORDERS_URL} from "../app/viewer/bordersAsset.ts"

describe("the landmass names", () => {
    it("were generated for the borders blob this build fetches", () => {
        expect(`/static/${LANDMASS_NAMES_BORDERS}`).toBe(BORDERS_URL)
    })
})
