import {BOARD, RUNS_URL} from "../gen/map/board.ts"
import {boardPositions, positionsHash} from "./board.ts"

// Where each tile is, by its id less one: latitude and longitude in degrees.
export type Places = {latitudes: Float32Array, longitudes: Float32Array}

// The map's convention, as its generator reads a point: latitude asin(y), longitude atan2(z, −x).
export function placesOf(positions: Float32Array): Places {
    const tiles = positions.length / 3
    const latitudes = new Float32Array(tiles)
    const longitudes = new Float32Array(tiles)
    for (let i = 0; i < tiles; i++) {
        const x = positions[i * 3], y = positions[i * 3 + 1], z = positions[i * 3 + 2]
        latitudes[i] = Math.asin(Math.min(1, Math.max(-1, y))) * 180 / Math.PI
        longitudes[i] = Math.atan2(z, -x) * 180 / Math.PI
    }
    return {latitudes, longitudes}
}

// The board's runs, rebuilt into positions; a hash that differs is told, and the positions kept: an ulp moves a tile
// by under a metre.
export function boardPlaces(runs: ArrayBuffer): Places {
    const positions = boardPositions(BOARD.lattice, BOARD.tiles, runs)
    const hash = positionsHash(positions)
    if (hash !== BOARD.positionsHash) console.error(`${BOARD.name}'s positions hash to ${hash} here, it says ${BOARD.positionsHash}`)
    return placesOf(positions)
}

let loading: Promise<Places> | undefined

// Read once a page load: the map does not move while it is open.
export function loadPlaces(): Promise<Places> {
    loading ??= fetch(RUNS_URL)
        .then(response => {
            if (!response.ok) throw new Error(`the board's runs answered ${response.status}`)
            return response.arrayBuffer()
        })
        .then(boardPlaces)
        .catch((err: unknown) => {
            loading = undefined
            throw err
        })
    return loading
}
