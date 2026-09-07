import {COORDINATES_URL} from "./coordinatesAsset.ts"
import {decodeCoordinates, type PointGeometryData} from "./coordinatesBinary.ts"

export type {PointGeometryData}

/**
 * Fetches the tile coordinates blob and views it as the typed arrays the
 * geometry needs. The data used to be a build-time JSON import, which put 25 MB
 * into the main chunk; fetching the binary keeps the chunk small and skips both
 * the JSON parse and the element-by-element copy into Float32Array.
 */
export async function loadPointGeometryData(signal?: AbortSignal): Promise<PointGeometryData> {
    const response = await fetch(COORDINATES_URL, {signal})
    if (!response.ok) {
        throw new Error(`Failed to fetch ${COORDINATES_URL}: ${response.status} ${response.statusText}`)
    }
    return decodeCoordinates(await response.arrayBuffer())
}
