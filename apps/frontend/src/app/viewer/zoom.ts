import {BoardScale, scaledZoom} from "../../domain/boardScale.ts"

export const MIN_ZOOM = 0.5

// The closest the camera comes on earth-1: a board's is where its tile is as wide on screen.
const EARTH_1_MAX_ZOOM = 50

export function maxZoomOf(scale: BoardScale): number {
    return scaledZoom(EARTH_1_MAX_ZOOM, scale)
}

export const RESTING_ZOOM = 1
