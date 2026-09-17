import {describe, expect, it} from "vitest"
import home from "../../index.html?raw"
import {COUNTRY_STORAGE_KEY} from "./countryPreference.ts"

describe("the home page", () => {
    it("sends a browser that has played before to the game, under the key the game writes", () => {
        expect(home).toContain(`localStorage.getItem("${COUNTRY_STORAGE_KEY}")`)
        expect(home).toContain(`location.replace("/play" + location.search + location.hash)`)
    })

    it("stays when the menu links back to it", () => {
        expect(home).toContain(`location.hash !== "#home"`)
        expect(home).toContain(`id="home"`)
    })
})
