import {Shot} from "./clip/camera.ts"

const LAT_PARAM = "lat"
const LON_PARAM = "lon"
const ZOOM_PARAM = "zoom"

// Degrees, and the camera's zoom: a Discord ping links to the attack it tells.
export function linkedShot(url: URL): Shot | undefined {
    const lat = numberOf(url, LAT_PARAM)
    const lon = numberOf(url, LON_PARAM)
    const zoom = numberOf(url, ZOOM_PARAM)
    if (lat === undefined || lon === undefined || zoom === undefined) return undefined
    if (Math.abs(lat) > 90 || Math.abs(lon) > 180 || zoom <= 0) return undefined

    const phi = lat * Math.PI / 180
    const lambda = lon * Math.PI / 180
    // As the tile positions: y is the north pole, and longitude 0 is on -x.
    return {
        direction: {x: -Math.cos(phi) * Math.cos(lambda), y: Math.sin(phi), z: Math.cos(phi) * Math.sin(lambda)},
        zoom,
    }
}

export function withoutLinkedShot(url: URL): URL {
    const stripped = new URL(url)
    for (const param of [LAT_PARAM, LON_PARAM, ZOOM_PARAM]) stripped.searchParams.delete(param)
    return stripped
}

function numberOf(url: URL, param: string): number | undefined {
    const value = url.searchParams.get(param)?.trim()
    if (!value) return undefined
    const number = Number(value)
    return Number.isFinite(number) ? number : undefined
}
