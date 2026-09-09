import {describe, expect, it} from "vitest"
import {countryFromTimeZone} from "./visitorCountry.ts"
import {Countries} from "../../domain/countries.ts"

describe("countryFromTimeZone", () => {
    it("maps a well-known zone to its country", () => {
        expect(countryFromTimeZone("Europe/Paris")).toEqual(Countries.get("fr"))
        expect(countryFromTimeZone("Asia/Tokyo")).toEqual(Countries.get("jp"))
    })

    it("returns undefined for a zone it does not know", () => {
        expect(countryFromTimeZone("Mars/Olympus_Mons")).toBeUndefined()
    })

    it("returns undefined when the environment reports no zone", () => {
        expect(countryFromTimeZone(undefined)).toBeUndefined()
        expect(countryFromTimeZone("")).toBeUndefined()
    })

    it("never returns a country the renderer cannot draw", () => {
        const zones = ["Europe/Paris", "Asia/Tokyo", "America/New_York", "Africa/Cairo",
            "Australia/Sydney", "Antarctica/McMurdo", "Pacific/Chatham", "Europe/Vatican"]
        for (const zone of zones) {
            const country = countryFromTimeZone(zone)
            if (country) expect(Countries.has(country.code), zone).toBe(true)
        }
    })
})
