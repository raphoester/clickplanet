import {BoardScale, scaledLength} from "./boardScale.ts"

// On earth-1 the six around a tile are at most 0.0044 away on the globe of radius 1, the next ring at least 0.0048: as
// many of its own spacings on another board.
const REACH = 0.0046

const TILE_BITS = 2 ** 22

// The tiles touching each tile, found in a grid of the positions: the tiles sorted by their cell.
export class Neighbours {
    private readonly keys: Uint32Array
    private readonly tiles: Uint32Array
    private readonly reachSquared: number
    // Twice the reach: the tiles within reach of a point are in the two cells around it on each axis.
    private readonly cell: number
    private readonly cellsPerAxis: number

    // positions holds x, y, z per tile, tile id - 1 first, on the board scale is of.
    constructor(private readonly positions: ArrayLike<number>, scale: BoardScale) {
        const reach = scaledLength(REACH, scale)
        this.reachSquared = reach * reach
        this.cell = 2 * reach
        this.cellsPerAxis = Math.ceil(2 / this.cell) + 2
        const count = Math.floor(positions.length / 3)
        const sorted = new Float64Array(count)
        for (let tile = 1; tile <= count; tile++) {
            const at = (tile - 1) * 3
            sorted[tile - 1] = this.keyOf(this.cellOf(positions[at]), this.cellOf(positions[at + 1]), this.cellOf(positions[at + 2])) * TILE_BITS + tile
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
        const ix = this.lowerCellOf(x), iy = this.lowerCellOf(y), iz = this.lowerCellOf(z)
        const near: number[] = []
        for (let dx = 0; dx < 2; dx++) for (let dy = 0; dy < 2; dy++) for (let dz = 0; dz < 2; dz++) {
            const key = this.keyOf(ix + dx, iy + dy, iz + dz)
            for (let i = this.firstOf(key); i < this.keys.length && this.keys[i] === key; i++) {
                const other = this.tiles[i]
                if (other === tile) continue
                const o = (other - 1) * 3
                const ox = this.positions[o] - x, oy = this.positions[o + 1] - y, oz = this.positions[o + 2] - z
                if (ox * ox + oy * oy + oz * oz <= this.reachSquared) near.push(other)
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

    private cellOf(coordinate: number): number {
        return Math.floor((coordinate + 1) / this.cell)
    }

    private lowerCellOf(coordinate: number): number {
        return Math.floor((coordinate + 1) / this.cell - 0.5)
    }

    private keyOf(x: number, y: number, z: number): number {
        return (x + 1) + this.cellsPerAxis * ((y + 1) + this.cellsPerAxis * (z + 1))
    }
}
