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

    // Google's brand verification reads this page, so the pitch to sign in must not say an account is needed.
    it("tells a visitor that signing in clicks faster, and that playing needs no account", () => {
        expect(home).toContain("Sign in, click 2× faster")
        expect(home).toContain("No account needed")
    })
})
