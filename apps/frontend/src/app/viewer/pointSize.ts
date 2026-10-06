import {MapView} from "../../domain/displaySettings.ts"

export function tilePointSize(zoom: number, viewportHeight: number): number {
    return zoom * 1.5 * (viewportHeight / 1000)
}

export function tileSpacing(zoom: number, viewportHeight: number): number {
    return zoom * 1.98 * (viewportHeight / 1000)
}

const SPREAD = 2.3 / 1.5

const COARSE_FROM = 5
const COARSE_UNTIL = 8

export function coarseHandover(tileSize: number): number {
    return Math.min(1, Math.max(0, (tileSize - COARSE_FROM) / (COARSE_UNTIL - COARSE_FROM)))
}

function handover(tileSize: number, view: MapView): number {
    return view === "tiles" ? 1 : coarseHandover(tileSize)
}

export function displayPointSize(zoom: number, viewportHeight: number, view: MapView): number {
    const base = tilePointSize(zoom, viewportHeight)
    return base * (SPREAD + (1 - SPREAD) * handover(base, view))
}

export function flagPaint(zoom: number, viewportHeight: number, view: MapView): number {
    return 1 - handover(tilePointSize(zoom, viewportHeight), view)
}

export const MAX_PICK_WINDOW = 257

export function pickWindowSize(pointSize: number): number {
    const needed = Math.ceil(pointSize) + 1
    const odd = needed % 2 === 0 ? needed + 1 : needed
    return Math.min(Math.max(odd, 3), MAX_PICK_WINDOW)
}
