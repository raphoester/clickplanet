import {Countries, Country} from "../../domain/countries.ts";
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
