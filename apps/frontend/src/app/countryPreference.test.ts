import {describe, expect, it} from "vitest"
import {FALLBACK_COUNTRY, parseStoredCountry, resolveCountry} from "./countryPreference.ts"
import {Countries} from "./countries.ts"
import {regions} from "./viewer/atlas.ts"

const FRANCE = Countries.get("fr")!
const JAPAN = Countries.get("jp")!

describe("parseStoredCountry", () => {
    it("reads back a country it wrote", () => {
        expect(parseStoredCountry(JSON.stringify(JAPAN))).toEqual(JAPAN)
    })

    it("returns the canonical country rather than trusting the stored name", () => {
        const tampered = JSON.stringify({code: "jp", name: "Not Japan"})
        expect(parseStoredCountry(tampered)).toEqual(JAPAN)
    })

    it("ignores an empty slot", () => {
        expect(parseStoredCountry(null)).toBeUndefined()
        expect(parseStoredCountry(undefined)).toBeUndefined()
        expect(parseStoredCountry("")).toBeUndefined()
    })

    it("ignores anything that is not valid JSON", () => {
        expect(parseStoredCountry("{not json")).toBeUndefined()
    })

    it("ignores JSON that is not a country-shaped object", () => {
        for (const raw of ['"fr"', "42", "null", "[]", "{}", '{"code":123}']) {
            expect(parseStoredCountry(raw), raw).toBeUndefined()
        }
    })

    /**
     * The crash this guards: a stored code with no sprite region made every
     * click throw, and the bad value was written straight back on each render,
     * so the app stayed broken across reloads until storage was cleared by hand.
     */
    it("rejects a country code we cannot render", () => {
        expect(parseStoredCountry(JSON.stringify({code: "zz", name: "Atlantis"}))).toBeUndefined()
    })
})

describe("resolveCountry", () => {
    it("prefers the stored country over the time zone", () => {
        expect(resolveCountry(JSON.stringify(JAPAN), "Europe/Paris")).toEqual(JAPAN)
    })

    it("falls back to the time zone when storage is empty", () => {
        expect(resolveCountry(null, "Asia/Tokyo")).toEqual(JAPAN)
    })

    it("falls back to the time zone when storage holds junk", () => {
        expect(resolveCountry("{not json", "Europe/Paris")).toEqual(FRANCE)
    })

    it("falls back to the default when the time zone is unknown or absent", () => {
        expect(resolveCountry(null, "Mars/Olympus_Mons")).toEqual(FALLBACK_COUNTRY)
        expect(resolveCountry(null, undefined)).toEqual(FALLBACK_COUNTRY)
    })

    /**
     * Everything downstream assumes the selected country can be drawn, so the
     * whole point of this function is that no input escapes without one.
     */
    it("always resolves to a country the renderer can draw", () => {
        const inputs = [null, undefined, "", "{}", "garbage", '{"code":"zz"}', JSON.stringify(JAPAN)]
        const zones = [undefined, "Mars/Olympus_Mons", "Asia/Tokyo", "Europe/Paris"]

        for (const stored of inputs) {
            for (const zone of zones) {
                const resolved = resolveCountry(stored, zone)
                expect(Countries.has(resolved.code), `${stored} / ${zone}`).toBe(true)
                expect(regions.has(resolved.code), `${stored} / ${zone}`).toBe(true)
            }
        }
    })
})
