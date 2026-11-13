import {dot, Point, pointOf} from "./clip/geometry.ts"

export type Adjacency = {of(tile: number): number[]}

export type FrontChanges = {on: number[], off: number[]}

const NONE: FrontChanges = {on: [], off: []}

// Every tile within rings steps of the centre, the centre left out, nearest ring first.
export function discOf(adjacency: Adjacency, centre: number, rings: number): number[] {
    const disc = discsOf(adjacency, [centre], rings)
    disc.delete(centre)
    return [...disc]
}

function discsOf(adjacency: Adjacency, centres: number[], rings: number): Set<number> {
    const seen = new Set(centres)
    let ring = centres
    for (let step = 0; step < rings; step++) {
        const next: number[] = []
        for (const tile of ring) {
            for (const near of adjacency.of(tile)) {
                if (seen.has(near)) continue
                seen.add(near)
                next.push(near)
            }
        }
        ring = next
    }
    return seen
}

// Each tile's neighbours found once: a widened front walks the same tiles over and over. Six at most a tile.
export function remembered(adjacency: Adjacency, tiles: number): Adjacency {
    const known = new Uint32Array((tiles + 1) * 7)
    return {
        of: (tile) => {
            const at = tile * 7
            if (known[at] === 0) {
                const near = adjacency.of(tile)
                if (near.length > 6) return near
                known[at] = near.length + 1
                known.set(near, at + 1)
            }
            return Array.from(known.subarray(at + 1, at + known[at]))
        },
    }
}

// The tiles a flag can push into: not its own, and touching one of its own (`acrossBridges`: the far end of a bridge too).
// Widened by a spread's rings, also every tile whose disc over the land touches one: the spread takes the tiles between.
export class Front {
    private flag: string | undefined
    private rings = 0
    private readonly edge = new Set<number>()
    private readonly lit = new Set<number>()

    // positions holds x, y, z per tile, tile id - 1 first.
    constructor(
        private readonly adjacency: Adjacency,
        private readonly ownerOf: (tile: number) => string | undefined,
        private readonly positions: ArrayLike<number>,
        private readonly land: Adjacency = adjacency,
    ) {}

    follow(flag: string): FrontChanges {
        const was = new Set(this.lit)
        this.forget()
        this.flag = flag
        for (let tile = 1; tile * 3 <= this.positions.length; tile++) {
            if (this.ownerOf(tile) !== flag) continue
            for (const next of this.adjacency.of(tile)) {
                if (this.ownerOf(next) !== flag) this.edge.add(next)
            }
        }
        this.lightAll()
        return {on: [...this.lit], off: [...was].filter((tile) => !this.lit.has(tile))}
    }

    widen(rings: number): FrontChanges {
        if (rings === this.rings) return NONE
        this.rings = rings
        if (this.flag === undefined) return NONE
        const was = new Set(this.lit)
        this.lit.clear()
        this.lightAll()
        return changesFrom(was, this.lit)
    }

    touch(tiles: readonly number[]): FrontChanges {
        if (this.flag === undefined || tiles.length === 0) return NONE
        const around = new Set<number>()
        for (const tile of tiles) {
            around.add(tile)
            for (const next of this.adjacency.of(tile)) around.add(next)
        }
        for (const tile of around) {
            if (this.onFront(tile, this.flag)) this.edge.add(tile)
            else this.edge.delete(tile)
        }

        const changes: FrontChanges = {on: [], off: []}
        for (const tile of this.widened(around)) {
            const lit = this.lights(tile)
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

    // A tile the flag holds, or one it can push into.
    reaches(tile: number): boolean {
        return this.flag !== undefined && (this.ownerOf(tile) === this.flag || this.lit.has(tile))
    }

    drop(): FrontChanges {
        const off = [...this.lit]
        this.forget()
        this.flag = undefined
        return {on: [], off}
    }

    nearestTo(point: Point, passes: (tile: number) => boolean = () => true): number | undefined {
        let nearest: number | undefined
        let closest = -Infinity
        for (const tile of this.lit) {
            if (!passes(tile)) continue
            const along = dot(pointOf(this.positions, tile), point)
            if (along > closest) {
                closest = along
                nearest = tile
            }
        }
        return nearest
    }

    facing(point: Point): boolean {
        for (const tile of this.lit) {
            if (dot(pointOf(this.positions, tile), point) > 0) return true
        }
        return false
    }

    private onFront(tile: number, flag: string): boolean {
        if (this.ownerOf(tile) === flag) return false
        return this.adjacency.of(tile).some((next) => this.ownerOf(next) === flag)
    }

    // A walk from a tile not the flag's never needs to cross the flag's land: an edge tile comes first.
    private lights(tile: number): boolean {
        if (this.ownerOf(tile) === this.flag) return false
        let found = false
        this.walk([tile], this.rings, (near) => found ||= this.edge.has(near))
        return found
    }

    private lightAll() {
        for (const tile of this.edge) this.lit.add(tile)
        this.walk([...this.edge], this.rings, (near) => {
            if (this.ownerOf(near) !== this.flag) this.lit.add(near)
            return false
        })
    }

    private widened(tiles: Set<number>): Set<number> {
        if (this.rings === 0) return tiles
        const near = new Set(tiles)
        for (const tile of discsOf(this.land, [...tiles], this.rings)) near.add(tile)
        return near
    }

    private walk(from: number[], rings: number, see: (tile: number) => boolean) {
        const seen = new Set(from)
        if (from.some(see)) return
        let ring = from
        for (let step = 0; step < rings && ring.length > 0; step++) {
            const next: number[] = []
            for (const tile of ring) {
                for (const near of this.land.of(tile)) {
                    if (seen.has(near)) continue
                    seen.add(near)
                    if (see(near)) return
                    if (this.ownerOf(near) !== this.flag) next.push(near)
                }
            }
            ring = next
        }
    }

    private forget() {
        this.edge.clear()
        this.lit.clear()
    }
}

function changesFrom(was: Set<number>, now: Set<number>): FrontChanges {
    return {on: [...now].filter((tile) => !was.has(tile)), off: [...was].filter((tile) => !now.has(tile))}
}
