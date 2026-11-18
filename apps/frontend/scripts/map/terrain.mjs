import {LAND, OPEN_SEA, SEA_BANDS, seaKind} from "../../src/domain/terrain.ts"

// The photo's colour each kind of land stands for, in Lab: the centres k-means (k = 8) found over earth-1's tiles
// (Nov 2026). Pinned, so a kind is the same colour on every board and keeps its number, which the palette is in.
export const PHOTO_COLOURS = {
    forest: [39.20, -30.73, 25.13],
    tundra: [62.66, -3.71, 4.86],
    boreal: [46.74, -19.55, 12.55],
    arid: [61.29, 10.62, 28.42],
    savanna: [58.41, -7.34, 32.67],
    grass: [50.48, -22.57, 31.89],
    sand: [78.98, 5.41, 22.69],
    snow: [97.87, -1.05, -0.56],
}

// Past this b*, a tile nearest snow is sand in the sun.
const YELLOW = 8
const SMOOTH_PASSES = 3
// A tile's own kind counts for this many neighbours in the vote.
const OWN_VOTE = 2

const SNOW = LAND.indexOf("snow")
const CENTRES = LAND.map((kind) => PHOTO_COLOURS[kind])

/**
 * The photo's colour at each tile, in Lab: nine pixels round it, spread wider east and west towards the poles,
 * where the equirectangular photo stretches.
 * @param {{data: Uint8Array, width: number, height: number, channels: number}} photo
 * @param {Float32Array} positions the tiles'
 * @returns {number[][]} [L, a, b] per tile
 */
export function tileColours({data, width, height, channels}, positions) {
    const labs = []
    for (let t = 0; t < positions.length / 3; t++) {
        const x = positions[t * 3], y = positions[t * 3 + 1], z = positions[t * 3 + 2]
        const lat = Math.asin(Math.max(-1, Math.min(1, y))) * 180 / Math.PI
        const lon = Math.atan2(z, -x) * 180 / Math.PI
        const cx = (lon + 180) / 360 * width
        const cy = (90 - lat) / 180 * height
        const stretch = Math.min(8, 1 / Math.max(Math.cos(lat * Math.PI / 180), 0.05))
        let r = 0, g = 0, b = 0
        for (let dy = -1; dy <= 1; dy++) {
            for (let dx = -1; dx <= 1; dx++) {
                const px = ((Math.floor(cx + dx * stretch) % width) + width) % width
                const py = Math.min(height - 1, Math.max(0, Math.floor(cy + dy)))
                const at = (py * width + px) * channels
                r += data[at]
                g += data[at + 1]
                b += data[at + 2]
            }
        }
        labs.push(labOf(r / 9, g / 9, b / 9))
    }
    return labs
}

/**
 * Each tile's kind of land: the one whose photo colour is nearest its own, but a yellow tile goes past snow to the
 * next nearest.
 * @param {number[][]} labs from `tileColours`
 * @returns {Uint8Array} an index of `LAND` per tile
 */
export function landKinds(labs) {
    return Uint8Array.from(labs, (lab) => {
        const kind = nearest(lab)
        return kind === SNOW && lab[2] > YELLOW ? nearest(lab, SNOW) : kind
    })
}

/**
 * The kind of each tile the photo shows as sea, from its tiles around. The photo's coast was cut to earth-1's tiles, so
 * a tile another board puts where earth-1 had sea reads the photo's blue, which is no kind of land. Wave by wave, each
 * such tile whose tiles around have a kind takes the one most of them have (the lower kind on a tie); one that none
 * reaches, an island earth-1 did not have, takes `fallback`'s.
 * @param {Uint8Array} kinds per tile, changed in place
 * @param {Uint8Array} astray per tile: 1 where the photo shows sea
 * @param {{at: Uint32Array, to: Uint32Array, tileOf: Int32Array}} lattice from `tileLattice`
 * @param {Uint32Array} points each tile's lattice point
 * @param {(tile: number) => number} fallback
 * @returns {{neighbours: number, fallback: number}} how many tiles took each
 */
export function kindsFromNeighbours(kinds, astray, {at, to, tileOf}, points, fallback) {
    const known = Uint8Array.from(astray, (off) => off ? 0 : 1)
    let pending = []
    for (let t = 0; t < astray.length; t++) if (astray[t]) pending.push(t)
    const votes = new Uint16Array(LAND.length)
    let taken = 0
    while (pending.length > 0) {
        const waiting = []
        const assigned = []
        for (const t of pending) {
            votes.fill(0)
            let any = false
            const v = points[t]
            for (let e = at[v]; e < at[v + 1]; e++) {
                const other = tileOf[to[e]]
                if (other < 0 || !known[other]) continue
                votes[kinds[other]]++
                any = true
            }
            if (!any) {
                waiting.push(t)
                continue
            }
            let best = 0
            for (let k = 1; k < LAND.length; k++) if (votes[k] > votes[best]) best = k
            assigned.push(t, best)
        }
        if (assigned.length === 0) break
        for (let k = 0; k < assigned.length; k += 2) {
            kinds[assigned[k]] = assigned[k + 1]
            known[assigned[k]] = 1
        }
        taken += assigned.length / 2
        pending = waiting
    }
    for (const t of pending) kinds[t] = fallback(t)
    return {neighbours: taken, fallback: pending.length}
}

/**
 * Speckle into patches: each tile takes the kind most of its neighbours and itself have, its own counting twice.
 * @param {Uint8Array} kinds per tile
 * @param {{at: Uint32Array, to: Uint32Array, tileOf: Int32Array}} lattice from `tileLattice`
 * @param {Uint32Array} points each tile's lattice point
 * @returns {Uint8Array}
 */
export function smoothed(kinds, {at, to, tileOf}, points, passes = SMOOTH_PASSES) {
    let current = kinds
    const votes = new Uint16Array(LAND.length)
    for (let pass = 0; pass < passes; pass++) {
        const next = new Uint8Array(current)
        for (let t = 0; t < current.length; t++) {
            votes.fill(0)
            votes[current[t]] += OWN_VOTE
            const v = points[t]
            for (let e = at[v]; e < at[v + 1]; e++) {
                const other = tileOf[to[e]]
                if (other >= 0) votes[current[other]]++
            }
            let best = current[t]
            for (let k = 0; k < LAND.length; k++) if (votes[k] > votes[best]) best = k
            next[t] = best
        }
        current = next
    }
    return current
}

/**
 * The board's terrain, by canonical id: each tile's kind of land, and each sea point's band, from how many lattice
 * steps it is from the nearest tile.
 * @param {Uint8Array} kinds per tile
 * @param {{at: Uint32Array, to: Uint32Array, tileOf: Int32Array}} lattice from `tileLattice`
 * @param {Uint32Array} ids each lattice point's canonical id
 * @returns {Uint8Array}
 */
export function terrainOf(kinds, {at, to, tileOf}, ids) {
    const terrain = new Uint8Array(ids.length).fill(OPEN_SEA)
    const reached = new Uint8Array(ids.length)
    let frontier = []
    for (let v = 0; v < ids.length; v++) {
        if (tileOf[v] < 0) continue
        terrain[ids[v]] = kinds[tileOf[v]]
        reached[v] = 1
        frontier.push(v)
    }
    for (let steps = 1; steps < SEA_BANDS && frontier.length > 0; steps++) {
        const next = []
        for (const v of frontier) {
            for (let e = at[v]; e < at[v + 1]; e++) {
                const sea = to[e]
                if (reached[sea]) continue
                reached[sea] = 1
                terrain[ids[sea]] = seaKind(steps)
                next.push(sea)
            }
        }
        frontier = next
    }
    return terrain
}

function nearest(lab, skip = -1) {
    let best = -1
    let bestDistance = Infinity
    CENTRES.forEach((centre, kind) => {
        const distance = (lab[0] - centre[0]) ** 2 + (lab[1] - centre[1]) ** 2 + (lab[2] - centre[2]) ** 2
        if (kind !== skip && distance < bestDistance) {
            bestDistance = distance
            best = kind
        }
    })
    return best
}

function linear(channel) {
    const c = channel / 255
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

/** CIE Lab under D65, from an sRGB colour of 0-255 channels. */
export function labOf(r, g, b) {
    const R = linear(r), G = linear(g), B = linear(b)
    const X = (0.4124 * R + 0.3576 * G + 0.1805 * B) / 0.95047
    const Y = 0.2126 * R + 0.7152 * G + 0.0722 * B
    const Z = (0.0193 * R + 0.1192 * G + 0.9505 * B) / 1.08883
    const f = (t) => t > 0.008856 ? Math.cbrt(t) : 7.787 * t + 16 / 116
    return [116 * f(Y) - 16, 500 * (f(X) - f(Y)), 200 * (f(Y) - f(Z))]
}
