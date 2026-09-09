import {Countries, Country} from "../domain/countries.ts";
import {countryFromTimeZone} from "./viewer/visitorCountry.ts";

export const COUNTRY_STORAGE_KEY = 'clickplanet-country'

export const FALLBACK_COUNTRY: Country = Countries.get("fr") ?? {code: "fr", name: "France"}

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

export function resolveCountry(
    stored: string | null | undefined,
    timeZone: string | undefined,
): Country {
    return parseStoredCountry(stored)
        ?? countryFromTimeZone(timeZone)
        ?? FALLBACK_COUNTRY
}
