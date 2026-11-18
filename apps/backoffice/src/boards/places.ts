import {Board, boardOf} from "./boards.ts"
import {boardPositions, positionsHash} from "./lattice.ts"

// Where each tile of a board is, by its id less one: latitude and longitude in degrees. A tile id is a place on its
// board alone, so the places carry it.
export type Places = {board: Board, latitudes: Float32Array, longitudes: Float32Array}

// The map's convention, as its generator reads a point: latitude asin(y), longitude atan2(z, −x).
export function placesOf(board: Board, positions: Float32Array): Places {
    const tiles = positions.length / 3
    const latitudes = new Float32Array(tiles)
    const longitudes = new Float32Array(tiles)
    for (let i = 0; i < tiles; i++) {
        const x = positions[i * 3], y = positions[i * 3 + 1], z = positions[i * 3 + 2]
        latitudes[i] = Math.asin(Math.min(1, Math.max(-1, y))) * 180 / Math.PI
        longitudes[i] = Math.atan2(z, -x) * 180 / Math.PI
    }
    return {board, latitudes, longitudes}
}

// A board's runs, rebuilt into positions; a hash that differs is told, and the positions kept: an ulp moves a tile
// by under a metre.
export function boardPlaces(board: Board, runs: ArrayBuffer): Places {
    const positions = boardPositions(board.lattice, board.tiles, runs)
    const hash = positionsHash(positions)
    if (hash !== board.positionsHash) console.error(`${board.name}'s positions hash to ${hash} here, it says ${board.positionsHash}`)
    return placesOf(board, positions)
}

const loading = new Map<string, Promise<Places>>()

// A board's places, read the first time an answer names it and kept for the page load: a board never changes. One
// the page does not carry is refused.
export function loadPlaces(name: string): Promise<Places> {
    let places = loading.get(name)
    if (places === undefined) {
        places = readPlaces(name)
        loading.set(name, places)
        places.catch(() => {
            if (loading.get(name) === places) loading.delete(name)
        })
    }
    return places
}

async function readPlaces(name: string): Promise<Places> {
    const {board, runsUrl} = boardOf(name)
    const response = await fetch(runsUrl)
    if (!response.ok) throw new Error(`${name}'s runs answered ${response.status}`)
    return boardPlaces(board, await response.arrayBuffer())
}
