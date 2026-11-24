import {Places} from "../boards/places.ts"

// A box of the planisphere in pixels: its top left corner in degrees, how many pixels a degree takes, and its size.
// `west` may lie past ±180: the world repeats east and west.
export type Box = {west: number, north: number, scale: number, width: number, height: number}

// The whole world at 4 pixels a degree: enough for the continents and the large islands.
export const WORLD_BOX: Box = {west: -180, north: 90, scale: 4, width: 1440, height: 720}

// A tile's reach from its middle, in steps of the lattice: past the half-diagonal of a cell (0.58), so the land has no
// holes where the lattice stretches.
const TILE_REACH = 0.65
// Near a pole a tile stretches across every longitude: its width stops at half the world.
const MIN_COS = 1e-3

// The tiles by one-degree cell, cell after cell: cell c holds order[starts[c]] to order[starts[c + 1]], each a tile's
// index. A box reads the cells it covers, not the whole board.
type Grid = {order: Uint32Array, starts: Uint32Array}

const COLUMNS = 360
const ROWS = 180

const grids = new WeakMap<Places, Grid>()

function cellOf(latitude: number, longitude: number): number {
    const row = Math.min(ROWS - 1, Math.max(0, Math.floor(latitude + 90)))
    const column = ((Math.floor(longitude + 180) % COLUMNS) + COLUMNS) % COLUMNS
    return row * COLUMNS + column
}

function gridOf(places: Places): Grid {
    const kept = grids.get(places)
    if (kept !== undefined) return kept
    const tiles = places.latitudes.length
    const cells = new Uint32Array(tiles)
    const starts = new Uint32Array(COLUMNS * ROWS + 1)
    for (let i = 0; i < tiles; i++) {
        cells[i] = cellOf(places.latitudes[i], places.longitudes[i])
        starts[cells[i] + 1]++
    }
    for (let c = 0; c < COLUMNS * ROWS; c++) starts[c + 1] += starts[c]
    const next = starts.slice(0, COLUMNS * ROWS)
    const order = new Uint32Array(tiles)
    for (let i = 0; i < tiles; i++) order[next[cells[i]]++] = i
    const grid = {order, starts}
    grids.set(places, grid)
    return grid
}

// The land in a box: 1 for each pixel a tile covers. A tile is an ellipse a little wider than the lattice's step, as
// wide as its longitude stretches, so the coasts are round and the land whole; a tile smaller than a pixel still
// covers the pixel it is on.
export function landOf(places: Places, box: Box): Uint8Array {
    const {west, north, scale, width, height} = box
    const land = new Uint8Array(width * height)
    const {order, starts} = gridOf(places)
    const radius = TILE_REACH * places.board.spacing * 180 / Math.PI
    const east = west + width / scale
    const south = north - height / scale
    const ry = radius * scale

    const rowFrom = Math.max(0, Math.floor(south - radius + 90))
    const rowTo = Math.min(ROWS - 1, Math.floor(north + radius + 90))
    for (let row = rowFrom; row <= rowTo; row++) {
        const poleward = Math.max(Math.abs(row - 90), Math.abs(row + 1 - 90))
        const reach = Math.min(180, radius / Math.max(Math.cos(poleward * Math.PI / 180), MIN_COS))
        const columnTo = Math.floor(east + reach)
        for (let column = Math.floor(west - reach); column <= columnTo; column++) {
            const cell = row * COLUMNS + ((((column + 180) % COLUMNS) + COLUMNS) % COLUMNS)
            const shift = column - (cell % COLUMNS - 180)
            for (let at = starts[cell]; at < starts[cell + 1]; at++) {
                const i = order[at]
                const latitude = places.latitudes[i]
                const cy = (north - latitude) * scale
                const cx = (places.longitudes[i] + shift - west) * scale
                const rx = Math.min(180, radius / Math.max(Math.cos(latitude * Math.PI / 180), MIN_COS)) * scale
                const top = Math.max(0, Math.floor(cy - ry))
                const bottom = Math.min(height - 1, Math.floor(cy + ry))
                for (let y = top; y <= bottom; y++) {
                    const dy = (y + 0.5 - cy) / ry
                    const half = Math.max(0.5, rx * Math.sqrt(Math.max(0, 1 - dy * dy)))
                    const left = Math.max(0, Math.floor(cx - half))
                    const right = Math.min(width, Math.ceil(cx + half))
                    if (right > left) land.fill(1, y * width + left, y * width + right)
                }
            }
        }
    }
    return land
}

const worlds = new WeakMap<Places, Uint8Array>()

// The whole world's land, drawn once a board.
export function worldLandOf(places: Places): Uint8Array {
    let land = worlds.get(places)
    if (land === undefined) {
        land = landOf(places, WORLD_BOX)
        worlds.set(places, land)
    }
    return land
}
