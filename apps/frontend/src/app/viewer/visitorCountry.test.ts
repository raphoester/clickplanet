import {describe, expect, it} from "vitest"
import {countryFromTimeZone} from "./visitorCountry.ts"
import {Countries} from "../countries.ts"

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

    /**
     * The time zone table is not the same data set as our country list, so a
     * zone can legitimately resolve to a code we have no flag for. Those have
     * to come back undefined rather than as an unrenderable country.
     */
    it("never returns a country the renderer cannot draw", () => {
        const zones = ["Europe/Paris", "Asia/Tokyo", "America/New_York", "Africa/Cairo",
            "Australia/Sydney", "Antarctica/McMurdo", "Pacific/Chatham", "Europe/Vatican"]
        for (const zone of zones) {
            const country = countryFromTimeZone(zone)
            if (country) expect(Countries.has(country.code), zone).toBe(true)
        }
    })
})
