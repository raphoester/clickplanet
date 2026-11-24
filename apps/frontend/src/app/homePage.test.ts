import {describe, expect, it} from "vitest"
import home from "../../index.html?raw"
import {COUNTRY_STORAGE_KEY} from "./countryPreference.ts"
import {withFlagCodes} from "../flagCodes.ts"
import {HANDED_REFERRER_KEY} from "../backends/landingStore.ts"
import {RECRUIT_TAG, RECRUITER_PARAM, SOURCE_TAG_PARAM, SOURCES} from "../domain/landing.ts"
import {SHARE_FLAG_PARAM} from "../domain/shareCard.ts"

describe("the home page", () => {
    it("shows itself to a first visit on the site", () => {
        expect(destinationOf("https://clickplanet.lol/", {})).toBeUndefined()
        expect(destinationOf("https://clickplanet.lol/", "no storage")).toBeUndefined()
        expect(destinationOf("https://clickplanet.lol/?f=de", {[COUNTRY_STORAGE_KEY]: "de"})).toBe("/play?f=de")
    })

    it("sends a browser that has played before to the game, under the key the game writes", () => {
        expect(destinationOf("https://clickplanet.lol/?gfx=halo#top", {[COUNTRY_STORAGE_KEY]: "fr"})).toBe("/play?gfx=halo#top")
    })

    it("stays when the menu links back to it", () => {
        expect(destinationOf("https://clickplanet.lol/#home", {[COUNTRY_STORAGE_KEY]: "fr"})).toBeUndefined()
    })

    it("tells a visitor that playing needs no account", () => {
        expect(home).toContain("No account needed")
    })
})

describe("a tracked address", () => {
    it("opens the game with its one letter as the source, for a first visit too", () => {
        expect(destinationOf("https://clickplanet.lol/t", {})).toBe("/play?s=t")
        expect(destinationOf("https://clickplanet.lol/Y/", "no storage")).toBe("/play?s=y")
        expect(destinationOf("https://clickplanet.lol/t", {[COUNTRY_STORAGE_KEY]: "fr"})).toBe("/play?s=t")
    })

    it("carries the flag and the name of the query, and its letter wins over a query's source", () => {
        expect(destinationOf("https://clickplanet.lol/r?f=pl&r=kutza", {})).toBe("/play?f=pl&r=kutza&s=r")
        expect(destinationOf("https://clickplanet.lol/v?f=pl&r=Ada-Lovelace&s=yt#top", {})).toBe("/play?f=pl&r=Ada-Lovelace&s=v#top")
    })

    it("hands the game the page before", () => {
        const handed: Record<string, string> = {}

        destinationOf("https://clickplanet.lol/t", {}, {referrer: "https://www.tiktok.com/", handed})

        expect(handed).toEqual({[HANDED_REFERRER_KEY]: "https://www.tiktok.com/"})
    })

    it("is only a letter it knows, alone: any other is the home page", () => {
        for (const path of ["/p", "/x", "/t/kutza", "/tt-"]) {
            expect(destinationOf(`https://clickplanet.lol${path}`, {}), path).toBeUndefined()
        }
    })

    it("knows the letters the game knows", () => {
        for (const letter of Object.values(SOURCES)) {
            expect(destinationOf(`https://clickplanet.lol/${letter}`, {})).toBe(`/play?s=${letter}`)
        }
    })
})

describe("an old player's address, already shared", () => {
    it("still opens the game on the flag, as a recruit link, for a first visit too", () => {
        expect(destinationOf("https://clickplanet.lol/de", {})).toBe("/play?f=de&s=r")
        expect(destinationOf("https://clickplanet.lol/de/", "no storage")).toBe("/play?f=de&s=r")
        expect(destinationOf("https://clickplanet.lol/DE", {[COUNTRY_STORAGE_KEY]: "fr"})).toBe("/play?f=de&s=r")
    })

    it("carries the name after the flag as it is written, its dashes and its case", () => {
        expect(destinationOf("https://clickplanet.lol/pl/kutza", {})).toBe("/play?f=pl&s=r&r=kutza")
        expect(destinationOf("https://clickplanet.lol/pl/Ada-Lovelace/", {})).toBe("/play?f=pl&s=r&r=Ada-Lovelace")
        expect(destinationOf("https://clickplanet.lol/fr/%C3%89mile", {})).toBe(`/play?f=fr&s=r&r=${encodeURIComponent("Émile")}`)
    })

    it("keeps the rest of the query and the hash, and takes the address's word over the query's", () => {
        expect(destinationOf("https://clickplanet.lol/pl/kutza?gfx=halo&s=yt&f=de&r=bob#top", {}))
            .toBe("/play?gfx=halo&s=r&f=pl&r=kutza#top")
        expect(destinationOf("https://clickplanet.lol/pl?r=bob", {})).toBe("/play?f=pl&s=r")
    })

    it("still opens the game on the flag when the name is not one, or storage is refused", () => {
        expect(destinationOf("https://clickplanet.lol/pl/%E0%A4%A", {})).toBe("/play?f=pl&s=r")
        expect(destinationOf("https://clickplanet.lol/pl/kutza", "no storage")).toBe("/play?f=pl&s=r&r=kutza")
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
        const sent = new URL(destinationOf("https://clickplanet.lol/pl/kutza", {})!, "https://clickplanet.lol").searchParams
        expect(sent.get(SHARE_FLAG_PARAM)).toBe("pl")
        expect(sent.get(SOURCE_TAG_PARAM)).toBe(RECRUIT_TAG)
        expect(sent.get(RECRUITER_PARAM)).toBe("kutza")
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
