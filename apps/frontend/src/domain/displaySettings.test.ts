import {describe, expect, it} from "vitest"
import {DEFAULT_DISPLAY_SETTINGS, parseDisplaySettings} from "./displaySettings.ts"

describe("parseDisplaySettings", () => {
    it("starts with the country flags and the plain globe", () => {
        expect(parseDisplaySettings(null)).toEqual({mapView: "flags", rendering: "plain"})
        expect(DEFAULT_DISPLAY_SETTINGS).toEqual({mapView: "flags", rendering: "plain"})
    })

    it("reads back what was saved", () => {
        const saved = {mapView: "tiles", rendering: "sharp"}
        expect(parseDisplaySettings(JSON.stringify(saved))).toEqual(saved)
    })

    it("falls back to the default for anything it cannot read", () => {
        expect(parseDisplaySettings("not json")).toEqual(DEFAULT_DISPLAY_SETTINGS)
        expect(parseDisplaySettings("\"tiles\"")).toEqual(DEFAULT_DISPLAY_SETTINGS)
        expect(parseDisplaySettings("null")).toEqual(DEFAULT_DISPLAY_SETTINGS)
    })

    it("keeps each setting it can read, and defaults the other", () => {
        expect(parseDisplaySettings(JSON.stringify({mapView: "tiles", rendering: "blurry"})))
            .toEqual({mapView: "tiles", rendering: "plain"})
        expect(parseDisplaySettings(JSON.stringify({rendering: "sharp"})))
            .toEqual({mapView: "flags", rendering: "sharp"})
    })
})
