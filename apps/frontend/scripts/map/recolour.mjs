const BLOCK = 64

const ENOUGH = 4

const FLAT = 12

/**
 * @param {Uint8Array} photo raw RGB, width * height * channels
 * @param {Float32Array} cover 1 on a tile's ground, 0 at sea, width * height
 * @param {{width: number, height: number, channels: number}} size
 * @returns {{pixels: Buffer, moved: number}} `moved` counts the pixels that changed visibly
 */
export function recolour(photo, cover, {width, height, channels}) {
    const size = {width, height, channels}
    const agrees = agreement(photo, cover, {
        land: plate(photo, cover, size),
        sea: plate(photo, cover.map((c) => 1 - c), size),
    }, size)

    const land = plate(photo, cover.map((c, i) => c * agrees[i]), size)
    const sea = plate(photo, cover.map((c, i) => (1 - c) * agrees[i]), size)

    const grounded = land.known && sea.known

    const pixels = Buffer.alloc(width * height * 3)
    let moved = 0

    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            const at = y * width + x
            const here = [land.at(x, y, 0), land.at(x, y, 1), land.at(x, y, 2)]
            const water = [sea.at(x, y, 0), sea.at(x, y, 1), sea.at(x, y, 2)]

            const apart = Math.hypot(here[0] - water[0], here[1] - water[1], here[2] - water[2])
            const correction = !grounded || apart < FLAT
                ? 0
                : cover[at] - opinionOf(photo, at, channels, here, water)

            let changed = false
            for (let channel = 0; channel < 3; channel++) {
                const was = photo[at * channels + channel]
                const value = Math.round(was + correction * (here[channel] - water[channel]))
                pixels[at * 3 + channel] = value < 0 ? 0 : value > 255 ? 255 : value
                if (Math.abs(pixels[at * 3 + channel] - was) > 2) changed = true
            }

            if (changed) moved++
        }
    }

    return {pixels, moved}
}

function agreement(photo, cover, first, {width, height, channels}) {
    const agrees = new Float32Array(width * height)

    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            const at = y * width + x
            const here = [first.land.at(x, y, 0), first.land.at(x, y, 1), first.land.at(x, y, 2)]
            const water = [first.sea.at(x, y, 0), first.sea.at(x, y, 1), first.sea.at(x, y, 2)]
            if (Math.hypot(here[0] - water[0], here[1] - water[1], here[2] - water[2]) < FLAT) continue

            const opinion = opinionOf(photo, at, channels, here, water)
            agrees[at] = 1 - Math.abs(cover[at] - opinion)
        }
    }

    return agrees
}

function opinionOf(photo, at, channels, here, water) {
    let along = 0
    let length = 0
    for (let channel = 0; channel < 3; channel++) {
        const axis = here[channel] - water[channel]
        along += (photo[at * channels + channel] - water[channel]) * axis
        length += axis * axis
    }
    const opinion = along / length
    return opinion < 0 ? 0 : opinion > 1 ? 1 : opinion
}

function plate(photo, weights, {width, height, channels}) {
    const columns = Math.ceil(width / BLOCK)
    const rows = Math.ceil(height / BLOCK)
    const sums = new Float64Array(columns * rows * 3)
    const held = new Float64Array(columns * rows)

    for (let y = 0; y < height; y++) {
        const row = Math.floor(y / BLOCK) * columns
        for (let x = 0; x < width; x++) {
            const at = y * width + x
            const weight = weights[at]
            if (weight <= 0) continue
            const block = row + Math.floor(x / BLOCK)
            held[block] += weight
            for (let channel = 0; channel < 3; channel++) {
                sums[block * 3 + channel] += weight * photo[at * channels + channel]
            }
        }
    }

    const values = new Float32Array(columns * rows * 3)
    const known = new Uint8Array(columns * rows)
    for (let block = 0; block < columns * rows; block++) {
        if (held[block] < ENOUGH) continue
        known[block] = 1
        for (let channel = 0; channel < 3; channel++) {
            values[block * 3 + channel] = sums[block * 3 + channel] / held[block]
        }
    }

    const grounded = fill(values, known, columns, rows)

    return {
        known: grounded,
        at(x, y, channel) {
            const u = x / BLOCK - 0.5
            const v = Math.min(Math.max(y / BLOCK - 0.5, 0), rows - 1)
            const i = Math.floor(u)
            const j = Math.floor(v)
            const fu = u - i
            const fv = v - j
            const j0 = Math.min(Math.max(j, 0), rows - 1)
            const j1 = Math.min(j0 + 1, rows - 1)
            const i0 = ((i % columns) + columns) % columns
            const i1 = (i0 + 1) % columns

            const top = read(values, j0 * columns + i0, channel)
                + (read(values, j0 * columns + i1, channel) - read(values, j0 * columns + i0, channel)) * fu
            const bottom = read(values, j1 * columns + i0, channel)
                + (read(values, j1 * columns + i1, channel) - read(values, j1 * columns + i0, channel)) * fu
            return top + (bottom - top) * fv
        },
    }
}

const read = (values, block, channel) => values[block * 3 + channel]

function fill(values, known, columns, rows) {
    let remaining = known.length - known.reduce((a, b) => a + b, 0)
    if (remaining === known.length) return false

    while (remaining > 0) {
        const next = Uint8Array.from(known)
        let filled = 0

        for (let j = 0; j < rows; j++) {
            for (let i = 0; i < columns; i++) {
                const block = j * columns + i
                if (known[block]) continue

                let weight = 0
                const sum = [0, 0, 0]
                for (const [di, dj] of [[-1, 0], [1, 0], [0, -1], [0, 1]]) {
                    const nj = j + dj
                    if (nj < 0 || nj >= rows) continue
                    const ni = ((i + di) % columns + columns) % columns
                    const neighbour = nj * columns + ni
                    if (!known[neighbour]) continue
                    weight++
                    for (let channel = 0; channel < 3; channel++) sum[channel] += values[neighbour * 3 + channel]
                }
                if (weight === 0) continue

                for (let channel = 0; channel < 3; channel++) values[block * 3 + channel] = sum[channel] / weight
                next[block] = 1
                filled++
            }
        }

        if (filled === 0) throw new Error("the plate has no colour anywhere to spread from")
        known.set(next)
        remaining -= filled
    }

    return true
}
