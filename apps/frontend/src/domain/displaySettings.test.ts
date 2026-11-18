import {describe, expect, it} from "vitest"
import {DEFAULT_DISPLAY_SETTINGS, parseDisplaySettings} from "./displaySettings.ts"

describe("parseDisplaySettings", () => {
    it("starts with each holder's flag, the countries and the sharp globe", () => {
        expect(parseDisplaySettings(null)).toEqual({mapView: "split", divisions: "countries", rendering: "sharp"})
        expect(DEFAULT_DISPLAY_SETTINGS).toEqual({mapView: "split", divisions: "countries", rendering: "sharp"})
    })

    it("reads back what was saved", () => {
        for (const mapView of ["big", "split", "tiles"]) {
            for (const divisions of ["countries", "provinces"]) {
                for (const rendering of ["sharp", "soft"]) {
                    const saved = {mapView, divisions, rendering}
                    expect(parseDisplaySettings(JSON.stringify(saved))).toEqual(saved)
                }
            }
        }
    })

    it("gives the settings saved before the provinces the countries", () => {
        expect(parseDisplaySettings(JSON.stringify({mapView: "big", rendering: "soft"})))
            .toEqual({mapView: "big", divisions: "countries", rendering: "soft"})
        expect(parseDisplaySettings(JSON.stringify({mapView: "big", divisions: "states", rendering: "soft"})).divisions)
            .toBe("countries")
    })

    it("gives the flags saved by the old switch the default look, and keeps the tiles", () => {
        expect(parseDisplaySettings(JSON.stringify({mapView: "flags", rendering: "soft"})).mapView).toBe("split")
        expect(parseDisplaySettings(JSON.stringify({mapView: "tiles", rendering: "soft"})).mapView).toBe("tiles")
    })

    it("gives the plain globe saved while HD was off by default the sharp globe, and keeps the map view", () => {
        expect(parseDisplaySettings(JSON.stringify({mapView: "tiles", rendering: "plain"})))
            .toEqual({mapView: "tiles", divisions: "countries", rendering: "sharp"})
    })

    it("falls back to the default for anything it cannot read", () => {
        expect(parseDisplaySettings("not json")).toEqual(DEFAULT_DISPLAY_SETTINGS)
        expect(parseDisplaySettings("\"tiles\"")).toEqual(DEFAULT_DISPLAY_SETTINGS)
        expect(parseDisplaySettings("null")).toEqual(DEFAULT_DISPLAY_SETTINGS)
    })

    it("keeps each setting it can read, and defaults the other", () => {
        expect(parseDisplaySettings(JSON.stringify({mapView: "tiles", rendering: "blurry"})))
            .toEqual({mapView: "tiles", divisions: "countries", rendering: "sharp"})
        expect(parseDisplaySettings(JSON.stringify({divisions: "provinces", rendering: "soft"})))
            .toEqual({mapView: "split", divisions: "provinces", rendering: "soft"})
    })
})
