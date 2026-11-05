// The tiles' hits as a flat map, each tile a cell of its own colour: land is where the tiles are, so it needs no outline
// of the world. Longitude 0 is the middle.
export type HeatLook = {land: string, cool: string, warm: string, hot: string}

// A tile is about 0.4° across: its cell is as tall, and as wide as its latitude stretches it.
const TILE_DEGREES = 0.4
const WIDEST_CELL = 64

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

export function drawHeat(canvas: HTMLCanvasElement, positions: Float32Array, hits: ReadonlyMap<number, number>, look: HeatLook) {
    const {width, height} = canvas
    const context = canvas.getContext("2d")
    if (!context) return
    const image = context.createImageData(width, height)
    const ramp = [rgbOf(look.land), rgbOf(look.cool), rgbOf(look.warm), rgbOf(look.hot)]
    const most = Math.log1p(scaleOf(hits))
    const pixelDegrees = 360 / width

    for (let tile = 1; tile * 3 <= positions.length; tile++) {
        const {latitude, longitude} = placeOf(positions, tile)
        const hit = hits.get(tile) ?? 0
        const colour = hit > 0 ? colourAt(ramp, 0.15 + 0.85 * Math.min(1, Math.log1p(hit) / most)) : ramp[0]
        const tall = Math.max(1, Math.round(TILE_DEGREES / pixelDegrees))
        const wide = Math.min(WIDEST_CELL, Math.max(tall, Math.round(tall / Math.max(Math.cos(latitude * Math.PI / 180), 1e-3))))
        const x = Math.floor(((longitude + 180) / 360) * width - wide / 2)
        const y = Math.floor(((90 - latitude) / 180) * height - tall / 2)
        fill(image, x, y, wide, tall, colour)
    }
    context.putImageData(image, 0, 0)
}

// The hottest tiles alone set the scale, so a few fought over all season do not leave the rest dark.
function scaleOf(hits: ReadonlyMap<number, number>): number {
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
