import {Countries, Country} from "../domain/countries.ts";
import {isOpenedCode} from "../domain/flags.ts";
import {countryFromTimeZone} from "./visitorCountry.ts";

export const COUNTRY_STORAGE_KEY = 'clickplanet-country'

export const FALLBACK_COUNTRY: Country = Countries.get("fr") ?? {code: "fr"}

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

    // A flag opened since the build is kept before the page reads the list: the held flag the server tells wins anyway.
    return Countries.get(code) ?? (isOpenedCode(code) ? {code} : undefined)
}

export function resolveCountry(
    stored: string | null | undefined,
    timeZone: string | undefined,
): Country {
    return parseStoredCountry(stored)
        ?? countryFromTimeZone(timeZone)
        ?? FALLBACK_COUNTRY
}
