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
        window.localStorage.setItem(DISPLAY_SETTINGS_STORAGE_KEY, JSON.stringify({mapView: "big", divisions: "provinces", rendering: "soft"}))

        const {mapView, divisions, rendering} = (await loaded()).getState()

        expect({mapView, divisions, rendering}).toEqual({mapView: "big", divisions: "provinces", rendering: "soft"})
    })

    it("save each change in this browser", async () => {
        const settings = await loaded()

        settings.getState().setMapView("tiles")
        settings.getState().setDivisions("provinces")
        settings.getState().setRendering("soft")

        expect(JSON.parse(window.localStorage.getItem(DISPLAY_SETTINGS_STORAGE_KEY)!))
            .toEqual({mapView: "tiles", divisions: "provinces", rendering: "soft"})
    })
})
