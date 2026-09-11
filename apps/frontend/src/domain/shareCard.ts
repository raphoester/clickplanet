import {Countries, Country} from "./countries.ts";

// The query parameter a shared link carries its country in: /?c=de. The share
// button that writes those links and the Worker that reads them both take the
// name from here, so the two cannot drift apart.
export const SHARE_COUNTRY_PARAM = "c"

const SITE_ORIGIN = "https://clickplanet.lol"

// Where `npm run og-countries` writes one card per country.
export const CARD_DIRECTORY = "static/og"

// The size scrapers ask for, and the size `og:image:width` / `og:image:height`
// in index.html claim. Every card — the generic one and all the per-country
// ones — is drawn at it, which is what lets the Worker rewrite the image URL
// without touching the dimensions beside it.
export const CARD_WIDTH = 1200
export const CARD_HEIGHT = 627

export type ShareCard = {
    title: string
    description: string
    url: string
    imageUrl: string
    imageAlt: string
}

/**
 * The countries a link can be shared for: the ones with both a flag and a
 * name. `npm run og-countries` draws a card for exactly this set, so a code in
 * here always has an image behind it.
 */
export const ShareableCountries: ReadonlyMap<string, Country> = Countries

/**
 * The country a `?c=` value names, or undefined for anything else — a missing
 * parameter, a malformed one, or a code we have no card for.
 *
 * **Nothing that fails this ever reaches a meta tag.** The card is built from
 * the country this hands back, never from the query string, so no value a
 * sharer invents can end up in the HTML we serve from our own domain.
 */
export function shareableCountry(raw: string | null | undefined): Country | undefined {
    if (typeof raw !== "string") return undefined

    const code = raw.trim().toLowerCase()
    if (!/^[a-z]{2}$/.test(code)) return undefined

    return ShareableCountries.get(code)
}

export function shareUrl(country: Country): string {
    return `${SITE_ORIGIN}/?${SHARE_COUNTRY_PARAM}=${country.code}`
}

export function cardImagePath(country: Country): string {
    return `${CARD_DIRECTORY}/${country.code}.jpg`
}

export function cardFor(country: Country): ShareCard {
    return {
        title: `${country.name} on ClickPlanet`,
        description: `See how much of the world ${country.name} holds — and take more of it, one click at a time.`,
        url: shareUrl(country),
        imageUrl: `${SITE_ORIGIN}/${cardImagePath(country)}`,
        imageAlt: `The flag of ${country.name} over the ClickPlanet globe`,
    }
}

/**
 * The card a request's query string asks for, or undefined to leave the page
 * alone. All three ways of asking for nothing — no parameter, an unknown code,
 * a malformed one — answer undefined, which is what keeps the generic card in
 * `index.html` as the fallback rather than a second copy of it in here.
 */
export function cardForQuery(search: string): ShareCard | undefined {
    const country = shareableCountry(new URLSearchParams(search).get(SHARE_COUNTRY_PARAM))
    return country && cardFor(country)
}

export type MetaOverride = {
    selector: string
    content: string
}

/**
 * Which tag each field of the card belongs in. The Worker walks this list and
 * rewrites `content`; the tags themselves stay in `index.html`, which is what
 * nginx serves in the local stack and what any crawler gets if the Worker is
 * out of the picture.
 *
 * `og:image:width` / `og:image:height` are deliberately not here: every card is
 * drawn at the same 1200x627 as the generic one, so the sizes already in
 * `index.html` stay true. Draw them at another size and they belong in here.
 */
export function metaOverrides(card: ShareCard): MetaOverride[] {
    return [
        {selector: 'meta[property="og:title"]', content: card.title},
        {selector: 'meta[property="og:description"]', content: card.description},
        {selector: 'meta[property="og:url"]', content: card.url},
        {selector: 'meta[property="og:image"]', content: card.imageUrl},
        {selector: 'meta[property="og:image:secure_url"]', content: card.imageUrl},
        {selector: 'meta[property="og:image:alt"]', content: card.imageAlt},
        {selector: 'meta[name="twitter:title"]', content: card.title},
        {selector: 'meta[name="twitter:description"]', content: card.description},
        {selector: 'meta[name="twitter:image"]', content: card.imageUrl},
        {selector: 'meta[name="twitter:image:alt"]', content: card.imageAlt},
    ]
}
