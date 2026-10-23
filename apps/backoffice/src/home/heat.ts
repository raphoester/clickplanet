import {Places} from "./places.ts"

// One cell per degree, for naming where a pointer is: row 0 is the north pole, column 0 is 180° west.
export const COLUMNS = 360
export const ROWS = 180

export type Cells = {tiles: Uint16Array, changes: Uint32Array}

export function cellOf(latitude: number, longitude: number): number {
    const row = Math.min(ROWS - 1, Math.max(0, Math.floor(90 - latitude)))
    const column = (Math.floor(longitude + 180) % COLUMNS + COLUMNS) % COLUMNS
    return row * COLUMNS + column
}

// `changes` is by tile id, which is the index of `places` plus one.
export function cellsOf(places: Places, changes: Uint32Array): Cells {
    const cells = {tiles: new Uint16Array(COLUMNS * ROWS), changes: new Uint32Array(COLUMNS * ROWS)}
    for (let i = 0; i < places.latitudes.length; i++) {
        const cell = cellOf(places.latitudes[i], places.longitudes[i])
        cells.tiles[cell]++
        cells.changes[cell] += changes[i + 1] ?? 0
    }
    return cells
}

export function busiestOf(cells: Cells): number | undefined {
    let busiest: number | undefined
    cells.changes.forEach((changes, cell) => {
        if (changes > 0 && (busiest === undefined || changes > cells.changes[busiest])) busiest = cell
    })
    return busiest
}

export function placeOf(cell: number): string {
    const latitude = 89.5 - Math.floor(cell / COLUMNS)
    const longitude = -179.5 + (cell % COLUMNS)
    return `${degreesOf(latitude, "N", "S")} ${degreesOf(longitude, "E", "W")}`
}

function degreesOf(value: number, positive: string, negative: string): string {
    return `${Math.abs(Math.trunc(value))}°${value >= 0 ? positive : negative}`
}

export type Picture = {width: number, height: number, land: Uint8Array, heat: Float32Array}

// A little more than the spacing between two tiles, in degrees, so no land is left with holes far north.
const TILE_DEGREES = 0.3
// Three box blurs are close to a gaussian: no square around a lone change.
const GLOW = 2
const PASSES = 3

// The map seen flat: each tile is drawn where it is, as wide as its longitude stretches, and what changed hands
// glows over the land around it.
export function pictureOf(places: Places, changes: Uint32Array, width: number): Picture {
    const height = width / 2
    const pixelDegrees = 360 / width
    const land = new Uint8Array(width * height)
    const raw = new Float32Array(width * height)

    for (let i = 0; i < places.latitudes.length; i++) {
        const latitude = places.latitudes[i]
        const y = Math.min(height - 1, Math.max(0, Math.floor(((90 - latitude) / 180) * height)))
        const x = ((places.longitudes[i] + 180) / 360) * width
        const half = Math.min(width / 2, Math.max(0.5,
            TILE_DEGREES / 2 / Math.max(Math.cos((latitude * Math.PI) / 180), 1e-3) / pixelDegrees))
        const changed = changes[i + 1] ?? 0

        for (let column = Math.floor(x - half); column < Math.ceil(x + half); column++) {
            const pixel = y * width + ((column % width) + width) % width
            land[pixel] = 1
            raw[pixel] += changed
        }
    }

    let heat = raw
    for (let pass = 0; pass < PASSES; pass++) heat = blur(heat, width, height, GLOW)
    return {width, height, land, heat}
}

// A box blur, round the world from east to west and stopped by the poles.
export function blur(values: Float32Array, width: number, height: number, radius: number): Float32Array {
    const across = new Float32Array(values.length)
    const span = 2 * radius + 1
    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            let sum = 0
            for (let d = -radius; d <= radius; d++) sum += values[y * width + ((x + d) % width + width) % width]
            across[y * width + x] = sum / span
        }
    }

    const down = new Float32Array(values.length)
    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            let sum = 0
            for (let d = -radius; d <= radius; d++) sum += across[Math.min(height - 1, Math.max(0, y + d)) * width + x]
            down[y * width + x] = sum / span
        }
    }
    return down
}

// Each pixel of land from 0 to 1, on a square root up to the hottest: a war stands out, a lone click is a tint.
export function levelsOf(picture: Picture): Float32Array {
    let highest = 0
    picture.heat.forEach((heat, pixel) => {
        if (picture.land[pixel] === 1) highest = Math.max(highest, heat)
    })

    const levels = new Float32Array(picture.heat.length)
    if (highest === 0) return levels

    picture.heat.forEach((heat, pixel) => {
        if (picture.land[pixel] === 1 && heat > 0) levels[pixel] = Math.sqrt(heat / highest)
    })
    return levels
}

export type Rgb = [number, number, number]

export function rgbOf(hex: string): Rgb {
    const value = hex.trim().replace(/^#/, "")
    const full = value.length === 3 ? [...value].map(digit => digit + digit).join("") : value
    const n = Number.parseInt(full, 16)
    return Number.isNaN(n) ? [0, 0, 0] : [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

export function colorAt(level: number, stops: Rgb[]): Rgb {
    const at = Math.min(Math.max(level, 0), 1) * (stops.length - 1)
    const i = Math.min(Math.floor(at), stops.length - 2)
    const t = at - i
    const [from, to] = [stops[i], stops[i + 1]]
    return [0, 1, 2].map(c => Math.round(from[c] + (to[c] - from[c]) * t)) as Rgb
}
