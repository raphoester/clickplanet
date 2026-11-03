import {dot, Point, pointOf} from "./clip/geometry.ts"

export type Adjacency = {of(tile: number): number[]}

export type FrontChanges = {on: number[], off: number[]}

const NONE: FrontChanges = {on: [], off: []}

// The tiles a flag can push into: not its own, and touching one of its own.
export class Front {
    private flag: string | undefined
    private readonly lit = new Set<number>()

    // positions holds x, y, z per tile, tile id - 1 first.
    constructor(
        private readonly adjacency: Adjacency,
        private readonly ownerOf: (tile: number) => string | undefined,
        private readonly positions: ArrayLike<number>,
    ) {}

    follow(flag: string): FrontChanges {
        const off = [...this.lit]
        this.lit.clear()
        this.flag = flag
        for (let tile = 1; tile * 3 <= this.positions.length; tile++) {
            if (this.ownerOf(tile) !== flag) continue
            for (const next of this.adjacency.of(tile)) {
                if (this.ownerOf(next) !== flag) this.lit.add(next)
            }
        }
        const on = [...this.lit]
        const kept = new Set(on)
        return {on, off: off.filter((tile) => !kept.has(tile))}
    }

    touch(tiles: readonly number[]): FrontChanges {
        if (this.flag === undefined || tiles.length === 0) return NONE
        const around = new Set<number>()
        for (const tile of tiles) {
            around.add(tile)
            for (const next of this.adjacency.of(tile)) around.add(next)
        }

        const changes: FrontChanges = {on: [], off: []}
        for (const tile of around) {
            const lit = this.onFront(tile, this.flag)
            if (lit === this.lit.has(tile)) continue
            if (lit) {
                this.lit.add(tile)
                changes.on.push(tile)
            } else {
                this.lit.delete(tile)
                changes.off.push(tile)
            }
        }
        return changes
    }

    nearestTo(point: Point): number | undefined {
        let nearest: number | undefined
        let closest = -Infinity
        for (const tile of this.lit) {
            const along = dot(pointOf(this.positions, tile), point)
            if (along > closest) {
                closest = along
                nearest = tile
            }
        }
        return nearest
    }

    private onFront(tile: number, flag: string): boolean {
        if (this.ownerOf(tile) === flag) return false
        return this.adjacency.of(tile).some((next) => this.ownerOf(next) === flag)
    }
}
