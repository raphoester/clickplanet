import {Countries, Country} from "./countries.ts";
import {countryFromTimeZone} from "./viewer/visitorCountry.ts";

export const COUNTRY_STORAGE_KEY = 'clickplanet-country'

/** Used when nothing else resolves. Asserted to exist by countryPreference.test.ts. */
export const FALLBACK_COUNTRY: Country = Countries.get("fr") ?? {code: "fr", name: "France"}

/**
 * Reads a country out of whatever localStorage happens to hold.
 *
 * Anything unparseable, malformed, or naming a country we have no flag for is
 * rejected. The renderer looks a sprite region up by country code on every
 * click and cannot draw a tile without one, so an unrenderable country read
 * back from storage used to brick the app permanently: the throw happened on
 * every click, and the bad value was written straight back on the next render.
 */
export function parseStoredCountry(raw: string | null | undefined): Country | undefined {
    if (!raw) return undefined

    let parsed: unknown
    try {
        parsed = JSON.parse(raw)
    } catch {
        return undefined
    }

    if (typeof parsed !== "object" || parsed === null) return undefined

    const code = (parsed as {code?: unknown}).code
    if (typeof code !== "string") return undefined

    return Countries.get(code)
}

/**
 * Picks the country to start with: whatever was stored, else a guess from the
 * visitor's time zone, else the fallback. The result is always renderable.
 */
export function resolveCountry(
    stored: string | null | undefined,
    timeZone: string | undefined,
): Country {
    return parseStoredCountry(stored)
        ?? countryFromTimeZone(timeZone)
        ?? FALLBACK_COUNTRY
}
