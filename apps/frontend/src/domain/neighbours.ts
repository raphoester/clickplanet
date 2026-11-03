// On the shipped map the six around a tile are at most 0.0044 away on the globe of radius 1, the next ring at least 0.0048.
const REACH = 0.0046

const REACH_SQUARED = REACH * REACH

// Twice the reach: the tiles within reach of a point are in the two cells around it on each axis.
const CELL = 2 * REACH

const CELLS_PER_AXIS = Math.ceil(2 / CELL) + 2

const TILE_BITS = 2 ** 22

// The tiles touching each tile, found in a grid of the positions: the tiles sorted by their cell.
export class Neighbours {
    private readonly keys: Uint32Array
    private readonly tiles: Uint32Array

    // positions holds x, y, z per tile, tile id - 1 first.
    constructor(private readonly positions: ArrayLike<number>) {
        const count = Math.floor(positions.length / 3)
        const sorted = new Float64Array(count)
        for (let tile = 1; tile <= count; tile++) {
            const at = (tile - 1) * 3
            sorted[tile - 1] = keyOf(cellOf(positions[at]), cellOf(positions[at + 1]), cellOf(positions[at + 2])) * TILE_BITS + tile
        }
        sorted.sort()

        this.keys = new Uint32Array(count)
        this.tiles = new Uint32Array(count)
        sorted.forEach((entry, i) => {
            this.keys[i] = Math.floor(entry / TILE_BITS)
            this.tiles[i] = entry % TILE_BITS
        })
    }

    of(tile: number): number[] {
        const at = (tile - 1) * 3
        const x = this.positions[at], y = this.positions[at + 1], z = this.positions[at + 2]
        const ix = lowerCellOf(x), iy = lowerCellOf(y), iz = lowerCellOf(z)
        const near: number[] = []
        for (let dx = 0; dx < 2; dx++) for (let dy = 0; dy < 2; dy++) for (let dz = 0; dz < 2; dz++) {
            const key = keyOf(ix + dx, iy + dy, iz + dz)
            for (let i = this.firstOf(key); i < this.keys.length && this.keys[i] === key; i++) {
                const other = this.tiles[i]
                if (other === tile) continue
                const o = (other - 1) * 3
                const ox = this.positions[o] - x, oy = this.positions[o + 1] - y, oz = this.positions[o + 2] - z
                if (ox * ox + oy * oy + oz * oz <= REACH_SQUARED) near.push(other)
            }
        }
        return near
    }

    private firstOf(key: number): number {
        let low = 0, high = this.keys.length
        while (low < high) {
            const middle = (low + high) >>> 1
            if (this.keys[middle] < key) low = middle + 1
            else high = middle
        }
        return low
    }
}

function cellOf(coordinate: number): number {
    return Math.floor((coordinate + 1) / CELL)
}

function lowerCellOf(coordinate: number): number {
    return Math.floor((coordinate + 1) / CELL - 0.5)
}

function keyOf(x: number, y: number, z: number): number {
    return (x + 1) + CELLS_PER_AXIS * ((y + 1) + CELLS_PER_AXIS * (z + 1))
}
