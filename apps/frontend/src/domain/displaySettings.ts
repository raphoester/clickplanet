// Zoomed out: the leader's flag over its whole landmass, each tile's holder's flag, or the tiles alone.
export type MapView = "big" | "split" | "tiles"

const MAP_VIEWS: readonly MapView[] = ["big", "split", "tiles"]

// "flags" was the old switch's on, from before "big" and "split": it reads as the default.
function isMapView(value: unknown): value is MapView {
    return MAP_VIEWS.some((view) => view === value)
}

export type Rendering = "sharp" | "plain"

export type DisplaySettings = {
    mapView: MapView
    rendering: Rendering
}

export const DISPLAY_SETTINGS_STORAGE_KEY = "clickplanet-display-settings"

export const DEFAULT_DISPLAY_SETTINGS: DisplaySettings = {mapView: "split", rendering: "plain"}

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
        mapView: isMapView(mapView) ? mapView : DEFAULT_DISPLAY_SETTINGS.mapView,
        rendering: rendering === "sharp" || rendering === "plain" ? rendering : DEFAULT_DISPLAY_SETTINGS.rendering,
    }
}
