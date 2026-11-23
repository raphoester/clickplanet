// A flag's footprint on a ground (a landmass or a province): the tiles that flag holds there. Each keeps the sums its
// frame is read from (see docs/zoomed-out-view.md), so a tile that changes hands moves two footprints in constant time.

export const NO_FOOTPRINT = 0

// Per footprint: the tile count, the sums of x, y, z and of xx, yy, zz, xy, xz, yz over its tiles' unit positions.
const SUMS = 10

export type FootprintFrame = {
    centre: [number, number, number]
    // Half its extent along the surface, east and north of its centre, in radians.
    halfEast: number
    halfNorth: number
}

export class Footprints {
    // Each tile's footprint, by tile index (tile id - 1), NO_FOOTPRINT when unowned or when none was free.
    readonly ofTile: Uint16Array
    // The footprints whose frame changed since the last `takeChanged`, each once: a flag a footprint, not a Set, as the
    // first read moves a million tiles.
    private changed: number[] = []
    private readonly isChanged: Uint8Array

    private readonly flagOfTile: Uint16Array
    private readonly idOf: Uint16Array
    private readonly keyOfId: Uint32Array
    private readonly sums: Float64Array
    private readonly free: number[] = []
    private next = 1

    constructor(
        private readonly unit: Float32Array,
        private readonly groundOf: Uint16Array,
        grounds: number,
        private readonly flags: number,
        readonly most: number,
    ) {
        const tiles = groundOf.length
        this.ofTile = new Uint16Array(tiles)
        this.flagOfTile = new Uint16Array(tiles)
        this.idOf = new Uint16Array(grounds * flags)
        this.keyOfId = new Uint32Array(most)
        this.sums = new Float64Array(most * SUMS)
        this.isChanged = new Uint8Array(most)
    }

    flagOf(tile: number): number {
        return this.flagOfTile[tile]
    }

    // Gives a tile (by index) to a flag number, 0 none; answers its footprint.
    set(tile: number, flag: number): number {
        if (flag === this.flagOfTile[tile]) return this.ofTile[tile]
        const was = this.ofTile[tile]
        if (was !== NO_FOOTPRINT) this.add(was, tile, -1)
        this.flagOfTile[tile] = flag
        const ground = this.groundOf[tile]
        const id = flag === 0 || ground === 0 ? NO_FOOTPRINT : this.idFor(ground * this.flags + flag)
        this.ofTile[tile] = id
        if (id !== NO_FOOTPRINT) this.add(id, tile, 1)
        return id
    }

    countOf(id: number): number {
        return this.sums[id * SUMS]
    }

    // Each footprint with a tile, once: its id, its ground and its flag number.
    forEachFootprint(visit: (id: number, ground: number, flag: number) => void) {
        for (let id = 1; id < this.next; id++) {
            if (this.sums[id * SUMS] <= 0) continue
            const key = this.keyOfId[id]
            visit(id, Math.floor(key / this.flags), key % this.flags)
        }
    }

    takeChanged(): number[] {
        const ids = this.changed
        this.changed = []
        for (const id of ids) this.isChanged[id] = 0
        return ids
    }

    // The frame a flag is painted in over its footprint, as the territory shader orients it: east along the parallel
    // of the centre, north across it. Undefined for a footprint with no tile.
    frameOf(id: number): FootprintFrame | undefined {
        const at = id * SUMS
        const n = this.sums[at]
        if (n <= 0) return undefined
        const mx = this.sums[at + 1] / n, my = this.sums[at + 2] / n, mz = this.sums[at + 3] / n
        const length = Math.hypot(mx, my, mz)
        // A footprint spread evenly round the globe has no middle: any centre is as good, and its extent is the whole.
        const centre: [number, number, number] = length > 1e-9 ? [mx / length, my / length, mz / length] : [0, 0, 1]
        const east = eastOf(centre)
        const north = cross(centre, east)
        return {
            centre,
            halfEast: halfExtent(this.sums, at, n, east, mx, my, mz),
            halfNorth: halfExtent(this.sums, at, n, north, mx, my, mz),
        }
    }

    private idFor(key: number): number {
        const known = this.idOf[key]
        if (known !== NO_FOOTPRINT) return known
        const id = this.free.pop() ?? (this.next < this.most ? this.next++ : NO_FOOTPRINT)
        if (id === NO_FOOTPRINT) return id
        this.idOf[key] = id
        this.keyOfId[id] = key
        return id
    }

    private add(id: number, tile: number, sign: 1 | -1) {
        const x = this.unit[tile * 3], y = this.unit[tile * 3 + 1], z = this.unit[tile * 3 + 2]
        const at = id * SUMS
        const s = this.sums
        s[at] += sign
        if (this.isChanged[id] === 0) {
            this.isChanged[id] = 1
            this.changed.push(id)
        }
        if (s[at] <= 0) {
            // Emptied: zeroed, not left to the drift of the sums, and freed for another flag or ground.
            s.fill(0, at, at + SUMS)
            this.idOf[this.keyOfId[id]] = NO_FOOTPRINT
            this.free.push(id)
            return
        }
        s[at + 1] += sign * x
        s[at + 2] += sign * y
        s[at + 3] += sign * z
        s[at + 4] += sign * x * x
        s[at + 5] += sign * y * y
        s[at + 6] += sign * z * z
        s[at + 7] += sign * x * y
        s[at + 8] += sign * x * z
        s[at + 9] += sign * y * z
    }
}

// Each tile's position on the unit sphere, 3 a tile.
export function unitPositions(positions: Float32Array): Float32Array {
    const unit = new Float32Array(positions.length)
    for (let i = 0; i < positions.length; i += 3) {
        const length = Math.hypot(positions[i], positions[i + 1], positions[i + 2]) || 1
        unit[i] = positions[i] / length
        unit[i + 1] = positions[i + 1] / length
        unit[i + 2] = positions[i + 2] / length
    }
    return unit
}

// As the territory fragment shader's paintedFlag: east along the parallel, any east at a pole.
export function eastOf([x, y, z]: readonly number[]): [number, number, number] {
    if (Math.abs(y) > 0.9999) return [1, 0, 0]
    const length = Math.hypot(z, x)
    return [z / length, 0, -x / length]
}

function cross([ax, ay, az]: readonly number[], [bx, by, bz]: readonly number[]): [number, number, number] {
    return [ay * bz - az * by, az * bx - ax * bz, ax * by - ay * bx]
}

// The half-width of an even spread with the tiles' variance along `axis` (√3 σ), as an arc.
function halfExtent(s: Float64Array, at: number, n: number, [ex, ey, ez]: readonly number[], mx: number, my: number, mz: number): number {
    const second = (s[at + 4] * ex * ex + s[at + 5] * ey * ey + s[at + 6] * ez * ez
        + 2 * (s[at + 7] * ex * ey + s[at + 8] * ex * ez + s[at + 9] * ey * ez)) / n
    const mean = mx * ex + my * ey + mz * ez
    const chord = Math.sqrt(3 * Math.max(0, second - mean * mean))
    return Math.asin(Math.min(1, chord))
}
