import {MapView} from "../../domain/displaySettings.ts"
import {BoardScale, scaledLength} from "../../domain/boardScale.ts"

// A tile's disc on screen, in CSS pixels: 1.5 at zoom 1 on a screen 1,000 high, on earth-1. A board's tiles as much wider
// as they are apart.
export function tilePointSize(zoom: number, viewportHeight: number, scale: BoardScale): number {
    return zoom * 1.5 * scale.length * (viewportHeight / 1000)
}

// The arc between two touching tiles on screen, in CSS pixels.
export function tileSpacing(zoom: number, viewportHeight: number, scale: BoardScale): number {
    return zoom * 1.98 * scale.length * (viewportHeight / 1000)
}

const COARSE_FROM = 5
const COARSE_UNTIL = 8

export function coarseHandover(tileSize: number): number {
    return Math.min(1, Math.max(0, (tileSize - COARSE_FROM) / (COARSE_UNTIL - COARSE_FROM)))
}

// The zoom until which the painted flags have the frame to themselves, before they blend into the tiles.
export function flagsZoomOf(viewportHeight: number, scale: BoardScale): number {
    return COARSE_FROM / tilePointSize(1, viewportHeight, scale)
}

// In orbit a tile is too small to aim at, so the pointer asks about a territory instead.
export function inOrbit(zoom: number, viewportHeight: number, scale: BoardScale): boolean {
    return tilePointSize(zoom, viewportHeight, scale) <= COARSE_FROM
}

// The zoom from which the tiles have the frame to themselves.
export function tilesZoomOf(viewportHeight: number, scale: BoardScale): number {
    return COARSE_UNTIL / tilePointSize(1, viewportHeight, scale)
}

// What the painted flags are framed by, the pointer lights in orbit and the lines from orbit add: the countries'
// landmasses, or their provinces.
export type Divisions = "countries" | "provinces"

// The ground a CSS pixel spans from which the provinces divide the map, in radians: PROVINCES_KM of the earth's 6,371.
// The province a point of land is in is about 500 km across on every board that has provinces (earth-2 and siege-1),
// so about 50 px from here on, and its flag reads; farther out, it is a few pixels and the map is noise.
const PROVINCES_KM = 10
const PROVINCES_FROM = 6371 / PROVINCES_KM
// The whole globe in view is the countries' on any screen, however tall.
const PROVINCES_LEAST_ZOOM = 1.25
// A zoom held on the line does not switch to and fro: the provinces come 5% past it and go 5% short of it.
const PROVINCES_MARGIN = 1.05

// The zoom from which the provinces divide the map, on a viewport this high in CSS pixels.
export function provincesZoomOf(viewportHeight: number): number {
    // The camera's frustum is 2 tall at zoom 1: a radian is half the viewport high.
    return Math.max(PROVINCES_LEAST_ZOOM, PROVINCES_FROM / (viewportHeight / 2))
}

// The divisions drawn at this zoom, by the ones drawn now.
export function divisionsAt(zoom: number, viewportHeight: number, shown: Divisions): Divisions {
    const from = provincesZoomOf(viewportHeight)
    const line = shown === "provinces" ? from / PROVINCES_MARGIN : from * PROVINCES_MARGIN
    return zoom >= line ? "provinces" : "countries"
}

// The provinces' frames fade in, or out, over this long from the frame the zoom crosses their line: a pinch that stops
// just past it rests on one framing, never on both, as a fade over a band of zooms would.
export const DIVISIONS_FADE_SECONDS = 0.25

// How far the provinces divide the map, 0 to 1, `seconds` after `share`, on its way to the divisions drawn; at once
// for a player who asks for less motion.
export function provinceShareAfter(share: number, divisions: Divisions, seconds: number, instant: boolean): number {
    const target = divisions === "provinces" ? 1 : 0
    if (instant) return target
    const step = Math.max(0, seconds) / DIVISIONS_FADE_SECONDS
    return target > share ? Math.min(target, share + step) : Math.max(target, share - step)
}

// A territory flown to is seen with its tiles on screen, and a small one no closer than tiles of this size.
const FLOWN_TILE_MOST = 28

export function flownZoomsOf(viewportHeight: number, scale: BoardScale): {least: number, most: number} {
    return {least: tilesZoomOf(viewportHeight, scale), most: FLOWN_TILE_MOST / tilePointSize(1, viewportHeight, scale)}
}

function handover(tileSize: number, view: MapView): number {
    return view === "tiles" ? 1 : coarseHandover(tileSize)
}

export function flagPaint(zoom: number, viewportHeight: number, view: MapView, scale: BoardScale): number {
    return 1 - handover(tilePointSize(zoom, viewportHeight, scale), view)
}

// How much nearer the camera an infected tile is drawn, so its pods lie over its neighbours: more than the depth between
// two of them, less than a tile at the limb behind the earth. 0.012 on earth-1 (the display vertex shader's virusLift).
export function virusLiftOf(scale: BoardScale): number {
    return scaledLength(0.012, scale)
}

// The least half-height a flag that does not lead its landmass covers it from, about a tile: 0.004 on earth-1 (the
// territory fragment shader's leastFlagRadius).
export function leastFlagRadiusOf(scale: BoardScale): number {
    return scaledLength(0.004, scale)
}
