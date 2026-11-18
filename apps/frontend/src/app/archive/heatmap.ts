import {BoardScale, scaledLength, scaleOf} from "../../domain/boardScale.ts"
import {latticeFaces, tileAt, type TileCells} from "../../domain/latticeCells.ts"

// The tiles' hits as a flat map, each tile a cell of its own colour: land is where the tiles are, so it needs no outline
// of the world. Longitude 0 is the middle.
export type HeatLook = {land: string, cool: string, warm: string, hot: string}

// A tile of earth-1 is about 0.4° across: its cell is as tall, and as wide as its latitude stretches it, up to 64 of
// the 4,096 pixels across. A board's cell is as big as its own tiles.
const TILE_DEGREES = 0.4
const WIDEST_CELL = 64

export function cellDegreesOf(scale: BoardScale): number {
    return scaledLength(TILE_DEGREES, scale)
}

// A board's tiles, where they are and the lattice cell each covers.
export type HeatBoard = {name: string, spacing: number, positions: Float32Array, cells: TileCells}

// An era's hits, by the tiles of its board.
export type HeatEra = {board: HeatBoard, hits: ReadonlyMap<number, number>}

// The hits of every era on one grid, the tiles of the finest board among them: each of its tiles takes the hits of the
// tile of each era's board its centre is on. The cells are geographic, so the eras add up; one era is as it is.
export function heatOf(eras: readonly HeatEra[]): {board: HeatBoard, hits: Map<number, number>} {
    const [first, ...rest] = eras
    if (first === undefined) throw new Error("a heatmap needs an era")
    const board = rest.reduce((finest, {board}) => board.spacing < finest.spacing ? board : finest, first.board)
    const hits = new Map<number, number>()
    const add = (tile: number, count: number) => hits.set(tile, (hits.get(tile) ?? 0) + count)
    const tiles = board.positions.length / 3
    for (const era of eras) {
        if (era.board.name === board.name) {
            era.hits.forEach((count, tile) => add(tile, count))
            continue
        }
        if (era.hits.size === 0) continue
        const faces = latticeFaces(era.board.cells.cols)
        for (let tile = 1; tile <= tiles; tile++) {
            const at = (tile - 1) * 3
            const under = tileAt(era.board.cells, faces, board.positions[at], board.positions[at + 1], board.positions[at + 2])
            const count = under === undefined ? undefined : era.hits.get(under)
            if (count) add(tile, count)
        }
    }
    return {board, hits}
}

export function lookOfThePage(): HeatLook {
    const style = getComputedStyle(document.documentElement)
    const token = (name: string) => style.getPropertyValue(name).trim()
    return {land: token("--panel-raised"), cool: token("--war"), warm: token("--gold"), hot: token("--cream")}
}

export function placeOf(positions: Float32Array, tile: number): {latitude: number, longitude: number} {
    const x = positions[(tile - 1) * 3], y = positions[(tile - 1) * 3 + 1], z = positions[(tile - 1) * 3 + 2]
    return {
        latitude: Math.asin(Math.max(-1, Math.min(1, y / Math.hypot(x, y, z)))) * 180 / Math.PI,
        longitude: Math.atan2(z, -x) * 180 / Math.PI,
    }
}

export function drawHeat(canvas: HTMLCanvasElement, {board, hits}: {board: HeatBoard, hits: ReadonlyMap<number, number>}, look: HeatLook) {
    const {positions} = board
    const scale = scaleOf(board)
    const {width, height} = canvas
    const context = canvas.getContext("2d")
    if (!context) return
    const image = context.createImageData(width, height)
    const ramp = [rgbOf(look.land), rgbOf(look.cool), rgbOf(look.warm), rgbOf(look.hot)]
    const most = Math.log1p(hottestOf(hits))
    const pixelDegrees = 360 / width

    for (let tile = 1; tile * 3 <= positions.length; tile++) {
        const {latitude, longitude} = placeOf(positions, tile)
        const hit = hits.get(tile) ?? 0
        const colour = hit > 0 ? colourAt(ramp, 0.15 + 0.85 * Math.min(1, Math.log1p(hit) / most)) : ramp[0]
        const tall = Math.max(1, Math.round(cellDegreesOf(scale) / pixelDegrees))
        const wide = Math.min(scaledLength(WIDEST_CELL, scale), Math.max(tall, Math.round(tall / Math.max(Math.cos(latitude * Math.PI / 180), 1e-3))))
        const x = Math.floor(((longitude + 180) / 360) * width - wide / 2)
        const y = Math.floor(((90 - latitude) / 180) * height - tall / 2)
        fill(image, x, y, wide, tall, colour)
    }
    context.putImageData(image, 0, 0)
}

// The hottest tiles alone set the scale, so a few fought over all season do not leave the rest dark.
function hottestOf(hits: ReadonlyMap<number, number>): number {
    const sorted = [...hits.values()].sort((a, b) => b - a)
    return Math.max(1, sorted[Math.floor(sorted.length * 0.005)] ?? 1)
}

// A cell across the map's edge goes on round to the other side.
function fill(image: ImageData, x: number, y: number, wide: number, tall: number, [r, g, b]: [number, number, number]) {
    for (let row = Math.max(0, y); row < Math.min(image.height, y + tall); row++) {
        for (let column = x; column < x + wide; column++) {
            const i = (row * image.width + ((column % image.width) + image.width) % image.width) * 4
            image.data[i] = r
            image.data[i + 1] = g
            image.data[i + 2] = b
            image.data[i + 3] = 255
        }
    }
}

function colourAt(ramp: [number, number, number][], level: number): [number, number, number] {
    const at = level * (ramp.length - 1)
    const low = ramp[Math.floor(at)], high = ramp[Math.min(ramp.length - 1, Math.floor(at) + 1)]
    const t = at - Math.floor(at)
    return [low[0] + (high[0] - low[0]) * t, low[1] + (high[1] - low[1]) * t, low[2] + (high[2] - low[2]) * t]
}

function rgbOf(colour: string): [number, number, number] {
    const hex = colour.replace("#", "")
    const full = hex.length === 3 ? hex.split("").map((c) => c + c).join("") : hex.slice(0, 6)
    return [parseInt(full.slice(0, 2), 16) || 0, parseInt(full.slice(2, 4), 16) || 0, parseInt(full.slice(4, 6), 16) || 0]
}
