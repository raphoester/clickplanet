import {COORDINATES_URL} from "../gen/map/coordinates.ts"

// Where each tile is, by its id less one: latitude and longitude in degrees.
export type Places = {latitudes: Float32Array, longitudes: Float32Array}

const MAGIC = "CPCO"
const VERSION = 1
const HEADER_BYTES = 12

// The map's own blob (/map/README.md): "CPCO", a version, the tile count, then x, y, z per tile as float32.
export function placesOf(buffer: ArrayBuffer): Places {
    const view = new DataView(buffer)
    const magic = String.fromCharCode(...new Uint8Array(buffer, 0, Math.min(4, buffer.byteLength)))
    if (magic !== MAGIC || view.getUint32(4, true) !== VERSION) throw new Error("not a coordinates file of version 1")

    const tiles = view.getUint32(8, true)
    if (buffer.byteLength < HEADER_BYTES + tiles * 12) throw new Error("the coordinates file is truncated")

    const latitudes = new Float32Array(tiles)
    const longitudes = new Float32Array(tiles)
    for (let i = 0; i < tiles; i++) {
        const at = HEADER_BYTES + i * 12
        const x = view.getFloat32(at, true), y = view.getFloat32(at + 4, true), z = view.getFloat32(at + 8, true)
        // The map's convention, as its generator reads a vertex.
        latitudes[i] = Math.asin(Math.min(1, Math.max(-1, y))) * 180 / Math.PI
        longitudes[i] = Math.atan2(z, -x) * 180 / Math.PI
    }

    return {latitudes, longitudes}
}

let loading: Promise<Places> | undefined

// Read once a page load: the map does not move while it is open.
export function loadPlaces(): Promise<Places> {
    loading ??= fetch(COORDINATES_URL)
        .then(response => {
            if (!response.ok) throw new Error(`the coordinates answered ${response.status}`)
            return response.arrayBuffer()
        })
        .then(placesOf)
        .catch((err: unknown) => {
            loading = undefined
            throw err
        })
    return loading
}
