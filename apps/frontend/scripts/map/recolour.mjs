// Moves a photograph's coastline onto the tile field's, without repainting the photograph.
//
// The globe's texture is a 4096x2048 render of a satellite mosaic, and it is the third thing that
// used to answer "sea or land". It cannot be an authority: at that size a one-tile island is three
// pixels, so the photo blends it into open water however carefully the polygons are drawn. It has to
// follow instead — a player must never see green with nothing to click on it, or a disc floating on
// open water.
//
// So the photo is corrected where it disagrees with the tile field, and **left alone everywhere
// else**, which is 98% of the globe:
//
//   out = photo + (cover - opinion) * (landColour - seaColour)
//
// `cover` is the tile field at full resolution, so a three-pixel island is corrected all the way.
// `opinion` is what the photo itself says, read off the same two colours: a pixel the colour of the
// land around it reads 1, one the colour of the water reads 0, and the ones between read between. So
// the correction is zero wherever the two already agree, and it is a real difference of colours
// rather than a palette somebody chose — `landColour` and `seaColour` are the photo's own local
// averages over the land and over the water, which follow latitude, depth and biome on their own.
//
// The first version of this was a frequency separation — `mix(sea, land, cover) + (photo - average)`
// — which is only the identity inside a block that is all land or all water. Along a coast it pushed
// the land further from the water in both directions and moved 37% of the image.
//
// **The two colours are found twice.** Taken straight off `cover`, a block whose only land is an
// island the photo drew as water learns that land here looks like water — and then leaves the island
// as it found it, which is the one case this exists for. So the second pass weights each pixel by
// whether the photo and the tile field already agree about it, and a block with no agreement left
// takes its colours from a neighbour that has some.

// The blur's block, in pixels. It has to be wide enough to hold both a coast's land and its water
// (so neither plate is empty along a coastline) and narrow enough to follow a biome. 64 is about six
// degrees, a few hundred kilometres.
const BLOCK = 64

// Below this a block has too little of one kind to average, and takes its colour from a neighbour
// that has some. Open ocean has no land for hundreds of blocks, which is what the fill is for.
const ENOUGH = 4

// Under this much difference between what land and water look like here, there is nothing to move a
// pixel along — under thick cloud, or on an ice shelf, where the two are the same white.
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

    // A plate with no block anywhere is a photo that never agreed with the tile field about land, or
    // never about water — a picture of nothing but ice. There is no direction to move a pixel along,
    // and a plate of black against one of blue would move every pixel a very long way.
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

// How much each pixel's own colour already says what the tile field says: 1 where they agree, 0
// where they do not. It is what the second pass weights by, so a block keeps only the pixels that
// are evidence and a block with none is filled from a neighbour.
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

// What the photo says about one pixel, on the line between what land looks like here and what water
// does: 1 at the land end, 0 at the water end, and the projection onto that line in between. A
// projection rather than a ratio of distances, so a colour that is neither — a cloud, a glacier —
// lands somewhere sensible instead of at 0.5 by default.
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

// One coarse plate: the photo's average colour per block over the pixels a weight selects, filled
// out into the blocks that had none, and read back with bilinear interpolation so nothing it
// contributes has an edge of its own.
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

    // Bilinear over the block centres. Longitude wraps; latitude clamps.
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

// Spreads the blocks that have a colour into the ones that do not, one ring at a time, until every
// block has one, and answers whether there was anything to spread from. A land plate over the middle
// of the Pacific is hundreds of blocks from any land, so this is what stops it being black there —
// and since nothing on open ocean reads the land plate (cover is 0, so the correction is zero
// whatever it says), what it settles on only has to be smooth, not right.
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
