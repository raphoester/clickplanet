import {existsSync, readFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {regions} from "../app/viewer/atlas.ts"
import {
    CARD_DIRECTORY,
    CARD_HEIGHT,
    CARD_WIDTH,
    cardFor,
    cardForQuery,
    metaOverrides,
    SHARE_COUNTRY_PARAM,
    ShareableCountries,
    shareableCountry,
    shareUrl,
} from "./shareCard.ts"
import {Countries} from "./countries.ts"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..")
const indexHtml = readFileSync(join(frontendRoot, "index.html"), "utf8")

const germany = Countries.get("de")!

describe("shareableCountry", () => {
    it("accepts a code from the country list", () => {
        expect(shareableCountry("de")).toEqual(germany)
    })

    it("accepts the casing and padding a shared link may arrive with", () => {
        expect(shareableCountry("DE")).toEqual(germany)
        expect(shareableCountry(" De ")).toEqual(germany)
    })

    it("refuses a code no country claims", () => {
        expect(shareableCountry("zz")).toBeUndefined()
    })

    it("refuses a code the atlas knows but no country is named for", () => {
        // The sprite atlas carries an EU flag; countries.json has no name to
        // title a card with, and no card is drawn for it.
        expect(regions.has("eu")).toBe(true)
        expect(shareableCountry("eu")).toBeUndefined()
    })

    it("refuses anything that is not two letters", () => {
        for (const raw of ["", "d", "deu", "d3", "--", "de fr", null, undefined]) {
            expect(shareableCountry(raw)).toBeUndefined()
        }
    })

    it("refuses markup, however it is dressed up", () => {
        for (const raw of ['"><script>alert(1)</script>', "de'", '" onload="x', "de&amp;"]) {
            expect(shareableCountry(raw)).toBeUndefined()
        }
    })
})

describe("cardForQuery", () => {
    it("has nothing to say about a link with no country on it", () => {
        expect(cardForQuery("")).toBeUndefined()
        expect(cardForQuery("?utm_source=discord")).toBeUndefined()
    })

    it("has nothing to say about a country it cannot draw", () => {
        expect(cardForQuery("?c=zz")).toBeUndefined()
        expect(cardForQuery('?c="><script>')).toBeUndefined()
    })

    it("reads the country past whatever else is on the link", () => {
        expect(cardForQuery("?utm_source=discord&c=de")).toEqual(cardFor(germany))
    })
})

describe("cardFor", () => {
    const card = cardFor(germany)

    it("names the country everywhere the card is read", () => {
        expect(card.title).toContain("Germany")
        expect(card.description).toContain("Germany")
        expect(card.imageAlt).toContain("Germany")
    })

    it("points at the country's own link, so each card is its own object", () => {
        expect(card.url).toBe(`https://clickplanet.lol/?${SHARE_COUNTRY_PARAM}=de`)
        expect(shareUrl(germany)).toBe(card.url)
    })

    it("points at an absolute image URL, which is all a scraper will fetch", () => {
        expect(card.imageUrl).toBe(`https://clickplanet.lol/${CARD_DIRECTORY}/de.jpg`)
    })
})

describe("metaOverrides", () => {
    const overrides = metaOverrides(cardFor(germany))

    it("rewrites every tag the generic card fills in", () => {
        expect(overrides.map((override) => override.selector)).toEqual([
            'meta[property="og:title"]',
            'meta[property="og:description"]',
            'meta[property="og:url"]',
            'meta[property="og:image"]',
            'meta[property="og:image:secure_url"]',
            'meta[property="og:image:alt"]',
            'meta[name="twitter:title"]',
            'meta[name="twitter:description"]',
            'meta[name="twitter:image"]',
            'meta[name="twitter:image:alt"]',
        ])
    })

    it("keeps the two image URLs in step", () => {
        const byTag = new Map(overrides.map(({selector, content}) => [selector, content]))

        expect(byTag.get('meta[property="og:image:secure_url"]'))
            .toBe(byTag.get('meta[property="og:image"]'))
        expect(byTag.get('meta[name="twitter:image"]'))
            .toBe(byTag.get('meta[property="og:image"]'))
    })

    // A selector that matches nothing rewrites nothing, silently: the card ends
    // up half generic and half German, which is worse than either.
    it("names tags index.html actually has", () => {
        for (const {selector} of overrides) {
            const attribute = selector.slice('meta['.length, -1)
            expect(indexHtml, selector).toContain(attribute)
        }
    })
})

describe("the cards on disk", () => {
    it("has one for every country a link can be shared for", () => {
        const missing = [...ShareableCountries.keys()]
            .filter((code) => !existsSync(join(frontendRoot, CARD_DIRECTORY, `${code}.jpg`)))

        expect(missing, "run `npm run og-countries`").toEqual([])
    })

    it("has a flag in the sprite atlas for every one of them", () => {
        const unflagged = [...ShareableCountries.keys()].filter((code) => !regions.has(code))

        expect(unflagged).toEqual([])
    })

})

// The Worker swaps the image URL and leaves og:image:width / og:image:height
// alone, which only holds while every card — the generic one and all 255 of
// these — is drawn at the one size the generators take from here.
it("draws every card at the size index.html claims", () => {
    expect(indexHtml).toContain(`<meta property="og:image:width" content="${CARD_WIDTH}"/>`)
    expect(indexHtml).toContain(`<meta property="og:image:height" content="${CARD_HEIGHT}"/>`)
})
