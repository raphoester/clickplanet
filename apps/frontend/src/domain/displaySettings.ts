export type MapView = "flags" | "tiles"

export type Rendering = "sharp" | "plain"

export type DisplaySettings = {
    mapView: MapView
    rendering: Rendering
}

export const DISPLAY_SETTINGS_STORAGE_KEY = "clickplanet-display-settings"

export const DEFAULT_DISPLAY_SETTINGS: DisplaySettings = {mapView: "flags", rendering: "plain"}

export function parseDisplaySettings(raw: string | null): DisplaySettings {
    if (raw === null) return DEFAULT_DISPLAY_SETTINGS

    let stored: unknown
    try {
        stored = JSON.parse(raw)
    } catch {
        return DEFAULT_DISPLAY_SETTINGS
    }
    if (typeof stored !== "object" || stored === null) return DEFAULT_DISPLAY_SETTINGS

    const {mapView, rendering} = stored as {mapView?: unknown, rendering?: unknown}
    return {
        mapView: mapView === "flags" || mapView === "tiles" ? mapView : DEFAULT_DISPLAY_SETTINGS.mapView,
        rendering: rendering === "sharp" || rendering === "plain" ? rendering : DEFAULT_DISPLAY_SETTINGS.rendering,
    }
}
