import {Place} from "../flags/pin.ts"
import {Box} from "./mapLand.ts"

// What the map shows: the place in its middle, how many CSS pixels a degree takes, and the frame's size. The longitude
// in the middle is never brought back within ±180, so a drag past the antimeridian goes on round the world.
export type View = {longitude: number, latitude: number, scale: number, width: number, height: number}

const KM_PER_DEGREE = 6371 * Math.PI / 180
// A capital opens with its country and the ones around it in the frame.
export const REGION_KM = 2000
// The closest the map goes: a few of the coarsest board's tiles across.
const CLOSEST_DEGREES = 1.5
// Far north a region is wider in degrees: up to this stretch.
const MIN_COS = 0.2

export function worldOf(width: number, height: number): View {
    return clampedOf({longitude: 0, latitude: 0, scale: 0, width, height})
}

export function regionOf(place: Place, width: number, height: number): View {
    const degrees = REGION_KM / (KM_PER_DEGREE * Math.max(Math.cos(place.latitude * Math.PI / 180), MIN_COS))
    return clampedOf({longitude: place.longitude, latitude: place.latitude, scale: width / degrees, width, height})
}

// From the whole world across the frame to CLOSEST_DEGREES; the poles at the frame's edges or inside it.
export function clampedOf(view: View): View {
    const scale = Math.min(view.width / CLOSEST_DEGREES, Math.max(view.width / 360, view.scale))
    const half = view.height / 2 / scale
    const latitude = half >= 90 ? 0 : Math.min(90 - half, Math.max(-90 + half, view.latitude))
    return {...view, scale, latitude}
}

export function isWorld(view: View): boolean {
    return view.width / view.scale >= 359
}

// Where a place is in the frame, in CSS pixels from its top left: the copy of the world nearest the middle.
export function pointOf(view: View, place: Place): [number, number] {
    const east = (((place.longitude - view.longitude + 180) % 360) + 360) % 360 - 180
    return [view.width / 2 + east * view.scale, view.height / 2 - (place.latitude - view.latitude) * view.scale]
}

// The place under a point of the frame, within ±90 and ±180.
export function placeAt(view: View, x: number, y: number): Place {
    const longitude = view.longitude + (x - view.width / 2) / view.scale
    const latitude = view.latitude - (y - view.height / 2) / view.scale
    return {latitude: Math.min(90, Math.max(-90, latitude)), longitude: ((((longitude + 180) % 360) + 360) % 360) - 180}
}

// Whether a point of the frame is on the globe, not past a pole.
export function onGlobe(view: View, y: number): boolean {
    return Math.abs(view.latitude - (y - view.height / 2) / view.scale) <= 90
}

// Zoomed by `factor` about a point of the frame, which stays over the same place.
export function zoomedAt(view: View, factor: number, x: number, y: number): View {
    const scale = clampedOf({...view, scale: view.scale * factor}).scale
    const longitude = view.longitude + (x - view.width / 2) * (1 / view.scale - 1 / scale)
    const latitude = view.latitude - (y - view.height / 2) * (1 / view.scale - 1 / scale)
    return clampedOf({...view, scale, longitude, latitude})
}

// Moved as a drag moves it: the map follows the pointer.
export function pannedBy(view: View, dx: number, dy: number): View {
    return clampedOf({...view, longitude: view.longitude - dx / view.scale, latitude: view.latitude + dy / view.scale})
}

// The same places across a frame of another size.
export function resizedTo(view: View, width: number, height: number): View {
    return clampedOf({...view, scale: view.scale * width / view.width, width, height})
}

// How far the frame spans at its middle's latitude, or undefined for the whole world.
export function kmAcross(view: View): number | undefined {
    if (isWorld(view)) return undefined
    return view.width / view.scale * KM_PER_DEGREE * Math.cos(view.latitude * Math.PI / 180)
}

// The frame and `margin` of its size on each side, at `pixelRatio` device pixels a CSS pixel, cut at the poles.
export function boxOf(view: View, pixelRatio: number, margin: number): Box {
    const scale = view.scale * pixelRatio
    const across = view.width / view.scale * (1 + 2 * margin)
    const north = Math.min(90, view.latitude + view.height / view.scale * (0.5 + margin))
    const south = Math.max(-90, view.latitude - view.height / view.scale * (0.5 + margin))
    return {
        west: view.longitude - across / 2,
        north,
        scale,
        width: Math.ceil(across * scale),
        height: Math.max(1, Math.ceil((north - south) * scale)),
    }
}

// Four decimals, about ten metres: what the page shows is what it sends.
export function roundedOf(place: Place): Place {
    const round = (degrees: number) => Math.round(degrees * 1e4) / 1e4
    return {latitude: round(place.latitude), longitude: round(place.longitude)}
}
