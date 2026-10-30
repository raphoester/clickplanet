import {describe, expect, it} from "vitest"
import {DEFAULT_DISPLAY_SETTINGS, parseDisplaySettings} from "./displaySettings.ts"

describe("parseDisplaySettings", () => {
    it("starts with each holder's flag and the plain globe", () => {
        expect(parseDisplaySettings(null)).toEqual({mapView: "split", rendering: "plain"})
        expect(DEFAULT_DISPLAY_SETTINGS).toEqual({mapView: "split", rendering: "plain"})
    })

    it("reads back what was saved", () => {
        for (const mapView of ["big", "split", "tiles"]) {
            const saved = {mapView, rendering: "sharp"}
            expect(parseDisplaySettings(JSON.stringify(saved))).toEqual(saved)
        }
    })

    it("gives the flags saved by the old switch the default look, and keeps the tiles", () => {
        expect(parseDisplaySettings(JSON.stringify({mapView: "flags", rendering: "plain"})).mapView).toBe("split")
        expect(parseDisplaySettings(JSON.stringify({mapView: "tiles", rendering: "plain"})).mapView).toBe("tiles")
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
            .toEqual({mapView: "split", rendering: "sharp"})
    })
})
