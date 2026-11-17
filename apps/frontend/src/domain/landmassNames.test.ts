import {describe, expect, it} from "vitest"
import {LANDMASS_NAMES_BORDERS} from "./landmassNames.ts"
import {BOARD} from "../app/viewer/boardAsset.ts"

describe("the landmass names", () => {
    it("were generated for the borders file of the board this build plays", () => {
        expect(LANDMASS_NAMES_BORDERS).toBe(BOARD.files.borders)
    })
})
