import {create} from "zustand"
import {
    DISPLAY_SETTINGS_STORAGE_KEY,
    DisplaySettings,
    MapView,
    parseDisplaySettings,
    Rendering,
} from "../../domain/displaySettings.ts"

type DisplaySettingsStore = DisplaySettings & {
    setMapView: (mapView: MapView) => void
    setRendering: (rendering: Rendering) => void
}

function readStoredSettings(): string | null {
    try {
        return window.localStorage.getItem(DISPLAY_SETTINGS_STORAGE_KEY)
    } catch {
        return null
    }
}

export const useDisplaySettings = create<DisplaySettingsStore>()((set) => ({
    ...parseDisplaySettings(readStoredSettings()),
    setMapView: (mapView) => set({mapView}),
    setRendering: (rendering) => set({rendering}),
}))

useDisplaySettings.subscribe(({mapView, rendering}) => {
    const settings: DisplaySettings = {mapView, rendering}
    try {
        window.localStorage.setItem(DISPLAY_SETTINGS_STORAGE_KEY, JSON.stringify(settings))
    } catch (e) {
        console.error("Could not persist the display settings", e)
    }
})
