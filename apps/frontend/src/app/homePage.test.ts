import {describe, expect, it} from "vitest"
import home from "../../index.html?raw"
import {CRAZYGAMES_HOST} from "../domain/crazyGames.ts"
import {COUNTRY_STORAGE_KEY} from "./countryPreference.ts"
import {FLAG_CODES_SLOT, withFlagCodes} from "../flagCodes.ts"
import {HANDED_REFERRER_KEY} from "../backends/landingStore.ts"
import {RECRUIT_TAG, RECRUITER_PARAM, SOURCE_TAG_PARAM} from "../domain/landing.ts"
import {SHARE_FLAG_PARAM} from "../domain/shareCard.ts"

describe("the home page", () => {
    it("is never shown on the CrazyGames page: a visitor goes straight to the game, with its invite", () => {
        expect(home).toContain(`location.hostname === "${CRAZYGAMES_HOST}"`)
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/?f=br&czy_invite=true`, {})).toBe("/play?f=br&czy_invite=true")
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/play/`, {})).toBe("/play")
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/#home`, {})).toBe("/play#home")
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/`, "no storage")).toBe("/play")
        expect(destinationOf(`https://${CRAZYGAMES_HOST}/pl/kutza`, {})).toBe("/play")
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

describe("a flag's address", () => {
    it("opens the game on the flag, as a recruit, for a first visit too", () => {
        expect(destinationOf("https://clickplanet.lol/de", {})).toBe("/play?f=de&s=rc")
        expect(destinationOf("https://clickplanet.lol/de/", "no storage")).toBe("/play?f=de&s=rc")
        expect(destinationOf("https://clickplanet.lol/DE", {[COUNTRY_STORAGE_KEY]: "fr"})).toBe("/play?f=de&s=rc")
        expect(destinationOf("https://clickplanet.lol/gb-sct", {})).toBe("/play?f=gb-sct&s=rc")
    })

    it("carries the name after the flag as it is written, its dashes and its case", () => {
        expect(destinationOf("https://clickplanet.lol/pl/kutza", {})).toBe("/play?f=pl&s=rc&r=kutza")
        expect(destinationOf("https://clickplanet.lol/pl/Ada-Lovelace/", {})).toBe("/play?f=pl&s=rc&r=Ada-Lovelace")
        expect(destinationOf("https://clickplanet.lol/fr/%C3%89mile", {})).toBe(`/play?f=fr&s=rc&r=${encodeURIComponent("Émile")}`)
    })

    it("keeps the rest of the query and the hash, and takes the address's word over the query's", () => {
        expect(destinationOf("https://clickplanet.lol/pl/kutza?gfx=halo&s=yt&f=de&r=bob#top", {}))
            .toBe("/play?gfx=halo&s=rc&f=pl&r=kutza#top")
        expect(destinationOf("https://clickplanet.lol/pl?r=bob", {})).toBe("/play?f=pl&s=rc")
    })

    it("still opens the game on the flag when the name is not one, or storage is refused", () => {
        expect(destinationOf("https://clickplanet.lol/pl/%E0%A4%A", {})).toBe("/play?f=pl&s=rc")
        expect(destinationOf("https://clickplanet.lol/pl/kutza", "no storage")).toBe("/play?f=pl&s=rc&r=kutza")
    })

    it("hands the game the page before, which a move to the game would otherwise lose", () => {
        const handed: Record<string, string> = {}

        destinationOf("https://clickplanet.lol/pl/kutza", {}, {referrer: "https://www.tiktok.com/", handed})

        expect(handed).toEqual({[HANDED_REFERRER_KEY]: "https://www.tiktok.com/"})
    })

    it("is only a flag the game knows: any other path is the home page, as before", () => {
        for (const path of ["/zz", "/zz/kutza", "/pl/kutza/more", "/play/", "/privacy-policy", "/p", "/pl%20"]) {
            expect(destinationOf(`https://clickplanet.lol${path}`, {}), path).toBeUndefined()
        }
        expect(destinationOf("https://clickplanet.lol/zz?f=de", {[COUNTRY_STORAGE_KEY]: "de"})).toBe("/play?f=de")
    })

    it("speaks the game's own words", () => {
        const script = headScript()
        expect(script).toContain(`query.set("${SHARE_FLAG_PARAM}", flag)`)
        expect(script).toContain(`query.set("${SOURCE_TAG_PARAM}", "${RECRUIT_TAG}")`)
        expect(script).toContain(`query.delete("${RECRUITER_PARAM}")`)
        expect(script).toContain(`sessionStorage.setItem("${HANDED_REFERRER_KEY}"`)
        expect(home).toContain(FLAG_CODES_SLOT)
    })
})

type Visit = {referrer: string, handed: Record<string, string>}

function destinationOf(
    href: string,
    stored: Record<string, string> | "no storage",
    visit: Visit = {referrer: "", handed: {}},
): string | undefined {
    const url = new URL(href)
    let replaced: string | undefined
    const location = {
        hostname: url.hostname,
        pathname: url.pathname,
        search: url.search,
        hash: url.hash,
        replace: (to: string) => {
            replaced = to
        },
    }
    const refuse = () => {
        throw new Error("storage refused")
    }
    const localStorage = {
        getItem: (key: string) => stored === "no storage" ? refuse() : stored[key] ?? null,
    }
    const sessionStorage = {
        setItem: (key: string, value: string) => {
            if (stored === "no storage") refuse()
            visit.handed[key] = value
        },
    }
    new Function("location", "localStorage", "sessionStorage", "document", headScript())(
        location, localStorage, sessionStorage, {referrer: visit.referrer})
    return replaced
}

function headScript(): string {
    const script = /<script>([\s\S]*?)<\/script>/.exec(withFlagCodes(home))
    if (!script) throw new Error("the home page has no inline script")
    return script[1]
}
