// The tiles' hits as a flat map that glows: land is where the tiles are, so it needs no outline of the world.
export type HeatLook = {land: string, cool: string, warm: string, hot: string}

const BLUR = 3

export function lookOfThePage(): HeatLook {
    const style = getComputedStyle(document.documentElement)
    const token = (name: string) => style.getPropertyValue(name).trim()
    return {land: token("--panel-raised"), cool: token("--war"), warm: token("--gold"), hot: token("--cream")}
}

export function drawHeat(canvas: HTMLCanvasElement, positions: Float32Array, hits: ReadonlyMap<number, number>, look: HeatLook) {
    const {width, height} = canvas
    const context = canvas.getContext("2d")
    if (!context) return
    const image = context.createImageData(width, height)
    const heat = new Float32Array(width * height)
    const land = rgbOf(look.land)

    for (let tile = 1; tile * 3 <= positions.length; tile++) {
        const at = pixelOf(positions, tile, width, height)
        for (const [dx, dy] of [[0, 0], [1, 0], [0, 1], [1, 1]]) paint(image, ((at.y + dy) % height) * width + ((at.x + dx) % width), land, 1)
        heat[at.y * width + at.x] += hits.get(tile) ?? 0
    }

    const blurred = blur(blur(heat, width, height), width, height)
    const most = Math.log1p(Math.max(1, ...sample(blurred)))
    const ramp = [rgbOf(look.land), rgbOf(look.cool), rgbOf(look.warm), rgbOf(look.hot)]
    for (let i = 0; i < blurred.length; i++) {
        if (blurred[i] <= 0) continue
        const level = Math.min(1, Math.log1p(blurred[i]) / most)
        paint(image, i, colourAt(ramp, level), Math.min(1, level * 2.5))
    }
    context.putImageData(image, 0, 0)
}

function pixelOf(positions: Float32Array, tile: number, width: number, height: number): {x: number, y: number} {
    const x = positions[(tile - 1) * 3], y = positions[(tile - 1) * 3 + 1], z = positions[(tile - 1) * 3 + 2]
    const latitude = Math.asin(Math.max(-1, Math.min(1, y / Math.hypot(x, y, z))))
    const longitude = Math.atan2(x, z)
    return {
        x: Math.min(width - 1, Math.floor(((longitude + Math.PI) / (2 * Math.PI)) * width)),
        y: Math.min(height - 1, Math.floor(((Math.PI / 2 - latitude) / Math.PI) * height)),
    }
}

// One pass of a box blur each way.
function blur(values: Float32Array, width: number, height: number): Float32Array {
    const across = new Float32Array(values.length)
    for (let y = 0; y < height; y++) {
        let sum = 0
        for (let x = -BLUR; x < width + BLUR; x++) {
            if (x + BLUR < width) sum += values[y * width + x + BLUR]
            if (x - BLUR - 1 >= 0) sum -= values[y * width + x - BLUR - 1]
            if (x >= 0 && x < width) across[y * width + x] = sum / (2 * BLUR + 1)
        }
    }
    const down = new Float32Array(values.length)
    for (let x = 0; x < width; x++) {
        let sum = 0
        for (let y = -BLUR; y < height + BLUR; y++) {
            if (y + BLUR < height) sum += across[(y + BLUR) * width + x]
            if (y - BLUR - 1 >= 0) sum -= across[(y - BLUR - 1) * width + x]
            if (y >= 0 && y < height) down[y * width + x] = sum / (2 * BLUR + 1)
        }
    }
    return down
}

// The hottest spots alone set the scale, so a few tiles fought over all season do not leave the rest dark.
function sample(values: Float32Array): number[] {
    const kept = [...values].filter((value) => value > 0).sort((a, b) => b - a)
    return kept.slice(Math.floor(kept.length * 0.002), Math.floor(kept.length * 0.002) + 1)
}

function colourAt(ramp: [number, number, number][], level: number): [number, number, number] {
    const at = level * (ramp.length - 1)
    const low = ramp[Math.floor(at)], high = ramp[Math.min(ramp.length - 1, Math.floor(at) + 1)]
    const t = at - Math.floor(at)
    return [low[0] + (high[0] - low[0]) * t, low[1] + (high[1] - low[1]) * t, low[2] + (high[2] - low[2]) * t]
}

function paint(image: ImageData, pixel: number, [r, g, b]: [number, number, number], alpha: number) {
    const i = pixel * 4
    const kept = image.data[i + 3] / 255
    const a = alpha + kept * (1 - alpha)
    image.data[i] = (r * alpha + image.data[i] * kept * (1 - alpha)) / Math.max(a, 1e-6)
    image.data[i + 1] = (g * alpha + image.data[i + 1] * kept * (1 - alpha)) / Math.max(a, 1e-6)
    image.data[i + 2] = (b * alpha + image.data[i + 2] * kept * (1 - alpha)) / Math.max(a, 1e-6)
    image.data[i + 3] = a * 255
}

function rgbOf(colour: string): [number, number, number] {
    const hex = colour.replace("#", "")
    const full = hex.length === 3 ? hex.split("").map((c) => c + c).join("") : hex.slice(0, 6)
    return [parseInt(full.slice(0, 2), 16) || 0, parseInt(full.slice(2, 4), 16) || 0, parseInt(full.slice(4, 6), 16) || 0]
}
