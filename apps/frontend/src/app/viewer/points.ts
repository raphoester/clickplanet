import {COORDINATES_URL} from "./coordinatesAsset.ts"
import {decodeCoordinates, type PointGeometryData} from "./coordinatesBinary.ts"

export type {PointGeometryData}

export async function loadPointGeometryData(signal?: AbortSignal): Promise<PointGeometryData> {
    const response = await fetch(COORDINATES_URL, {signal})
    if (!response.ok) {
        throw new Error(`Failed to fetch ${COORDINATES_URL}: ${response.status} ${response.statusText}`)
    }
    return decodeCoordinates(await response.arrayBuffer())
}
