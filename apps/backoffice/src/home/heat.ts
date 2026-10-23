// The grid GetActivity answers: row 0 is the north pole, column 0 is 180° west, one cell per degree.
export type Grid = {columns: number, rows: number, land: Uint8Array, changes: number[]}

export type Heat = {
    levels: Float32Array
    calmest: number
    busiest: number
    hottest: number | undefined
}

// Changes per tile, so a cell far north, which holds fewer tiles, is not read as calmer.
export function heatOf(grid: Grid): Heat {
    const rates = grid.changes.map((changes, cell) => changes / Math.max(grid.land[cell] ?? 0, 1))

    let calmest = Infinity, busiest = 0, hottest: number | undefined
    rates.forEach((rate, cell) => {
        if (rate <= 0) return
        calmest = Math.min(calmest, rate)
        if (rate > busiest) {
            busiest = rate
            hottest = cell
        }
    })

    const levels = new Float32Array(rates.length)
    if (busiest === 0) return {levels, calmest: 0, busiest: 0, hottest: undefined}

    const range = Math.log(busiest / calmest)
    rates.forEach((rate, cell) => {
        if (rate > 0) levels[cell] = range === 0 ? 1 : Math.log(rate / calmest) / range
    })

    return {levels, calmest, busiest, hottest}
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

export function placeOf(cell: number, columns: number): string {
    const latitude = 89.5 - Math.floor(cell / columns)
    const longitude = -179.5 + (cell % columns)
    return `${degreesOf(latitude, "N", "S")} ${degreesOf(longitude, "E", "W")}`
}

function degreesOf(value: number, positive: string, negative: string): string {
    return `${Math.abs(Math.trunc(value))}°${value >= 0 ? positive : negative}`
}
