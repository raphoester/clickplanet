// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {DISPLAY_SETTINGS_STORAGE_KEY} from "../../domain/displaySettings.ts"

afterEach(() => {
    window.localStorage.clear()
    vi.resetModules()
})

const loaded = async () => (await import("./displaySettingsStore.ts")).useDisplaySettings

describe("the display settings", () => {
    it("start from what this browser saved", async () => {
        window.localStorage.setItem(DISPLAY_SETTINGS_STORAGE_KEY, JSON.stringify({mapView: "big", rendering: "soft"}))

        const settings = (await loaded()).getState()

        expect({mapView: settings.mapView, rendering: settings.rendering}).toEqual({mapView: "big", rendering: "soft"})
    })

    it("save each change in this browser", async () => {
        const settings = await loaded()

        settings.getState().setMapView("tiles")
        settings.getState().setRendering("soft")

        expect(JSON.parse(window.localStorage.getItem(DISPLAY_SETTINGS_STORAGE_KEY)!)).toEqual({mapView: "tiles", rendering: "soft"})
    })
})
