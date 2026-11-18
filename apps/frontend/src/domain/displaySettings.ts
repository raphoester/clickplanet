// Zoomed out: the leader's flag over its whole landmass, each tile's holder's flag, or the tiles alone.
export type MapView = "big" | "split" | "tiles"

const MAP_VIEWS: readonly MapView[] = ["big", "split", "tiles"]

// "flags" was the old switch's on, from before "big" and "split": it reads as the default.
function isMapView(value: unknown): value is MapView {
    return MAP_VIEWS.some((view) => view === value)
}

// What the pointer lights and each big flag is framed by, and the lines seen from orbit: countries, or their provinces.
export type Divisions = "countries" | "provinces"

// "plain" was saved by every page while HD was off by default: it reads as the default.
export type Rendering = "sharp" | "soft"

export type DisplaySettings = {
    mapView: MapView
    divisions: Divisions
    rendering: Rendering
}

export const DISPLAY_SETTINGS_STORAGE_KEY = "clickplanet-display-settings"

export const DEFAULT_DISPLAY_SETTINGS: DisplaySettings = {mapView: "split", divisions: "countries", rendering: "sharp"}

export function parseDisplaySettings(raw: string | null): DisplaySettings {
    if (raw === null) return DEFAULT_DISPLAY_SETTINGS

    let stored: unknown
    try {
        stored = JSON.parse(raw)
    } catch {
        return DEFAULT_DISPLAY_SETTINGS
    }
    if (typeof stored !== "object" || stored === null) return DEFAULT_DISPLAY_SETTINGS

    const {mapView, divisions, rendering} = stored as {mapView?: unknown, divisions?: unknown, rendering?: unknown}
    return {
        mapView: isMapView(mapView) ? mapView : DEFAULT_DISPLAY_SETTINGS.mapView,
        divisions: divisions === "countries" || divisions === "provinces" ? divisions : DEFAULT_DISPLAY_SETTINGS.divisions,
        rendering: rendering === "sharp" || rendering === "soft" ? rendering : DEFAULT_DISPLAY_SETTINGS.rendering,
    }
}
