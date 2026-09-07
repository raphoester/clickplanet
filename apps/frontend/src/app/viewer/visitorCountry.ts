import {Countries, Country} from "../countries.ts";
import timezonesData from "moment-timezone/data/meta/latest.json"

const timezones = timezonesData as {
    countries: {
        [key: string]: {
            name: string
        }
    }
    zones: {
        [key: string]: {
            countries: string[]
        }
    }
}

/**
 * Best guess at the visitor's country from their IANA time zone.
 *
 * Returns undefined rather than a made-up country when the zone is unknown or
 * maps to a code we have no flag for: every caller has to end up with a country
 * the atlas can actually render, and inventing one here only moves the failure
 * to the click handler.
 */
export function countryFromTimeZone(timeZone: string | undefined): Country | undefined {
    if (!timeZone) return undefined

    const code = timezones.zones[timeZone]?.countries?.[0]
    if (!code) return undefined

    return Countries.get(code.toLowerCase())
}

export function currentTimeZone(): string | undefined {
    try {
        return Intl.DateTimeFormat().resolvedOptions().timeZone
    } catch {
        return undefined
    }
}
