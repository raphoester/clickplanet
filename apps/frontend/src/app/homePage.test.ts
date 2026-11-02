import {describe, expect, it} from "vitest"
import home from "../../index.html?raw"
import {CRAZYGAMES_HOST} from "../domain/crazyGames.ts"
import {COUNTRY_STORAGE_KEY} from "./countryPreference.ts"

describe("the home page", () => {
    it("is never shown on the CrazyGames page: a visitor goes straight to the game, with its invite", () => {
        expect(home).toContain(`location.hostname === "${CRAZYGAMES_HOST}"`)
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/?f=br&czy_invite=true`, {})).toBe("/play?f=br&czy_invite=true")
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/play/`, {})).toBe("/play")
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/#home`, {})).toBe("/play#home")
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/`, "no storage")).toBe("/play")
    })

    it("shows itself to a first visit on the site", () => {
        expect(destinationOf("https://clickplanet.lol/", {})).toBeUndefined()
        expect(destinationOf("https://clickplanet.lol/", "no storage")).toBeUndefined()
        expect(destinationOf("https://clickplanet.lol/?f=de", {[COUNTRY_STORAGE_KEY]: "de"})).toBe("/play?f=de")
    })

    it("sends a browser that has played before to the game, under the key the game writes", () => {
        expect(home).toContain(`localStorage.getItem("${COUNTRY_STORAGE_KEY}")`)
        expect(home).toContain(`location.replace("/play" + location.search + location.hash)`)
    })

    it("stays when the menu links back to it", () => {
        expect(home).toContain(`location.hash !== "#home"`)
        expect(home).toContain(`id="home"`)
    })

    it("tells a visitor that playing needs no account", () => {
        expect(home).toContain("No account needed")
    })
})

function destinationOf(href: string, stored: Record<string, string> | "no storage"): string | undefined {
    const url = new URL(href)
    let replaced: string | undefined
    const location = {
        hostname: url.hostname,
        search: url.search,
        hash: url.hash,
        replace: (to: string) => {
            replaced = to
        },
    }
    const localStorage = {
        getItem: (key: string) => {
            if (stored === "no storage") throw new Error("storage refused")
            return stored[key] ?? null
        },
    }
    new Function("location", "localStorage", headScript())(location, localStorage)
    return replaced
}

function headScript(): string {
    const script = /<script>([\s\S]*?)<\/script>/.exec(home)
    if (!script) throw new Error("the home page has no inline script")
    return script[1]
}
