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
