import type {Adjacency} from "./front.ts"

const NONE: readonly number[] = []

export type Bridge = readonly [number, number]

// Each crossing of the sea the map draws, by its two end tiles.
export class Bridges {
    private readonly ends = new Map<number, number[]>()

    constructor(readonly all: readonly Bridge[]) {
        for (const [a, b] of all) {
            this.link(a, b)
            this.link(b, a)
        }
    }

    across(tile: number): readonly number[] {
        return this.ends.get(tile) ?? NONE
    }

    private link(from: number, to: number) {
        const ends = this.ends.get(from)
        if (ends) ends.push(to)
        else this.ends.set(from, [to])
    }
}

// The land's neighbours, and the far end of each bridge from a tile as one more.
export function acrossBridges(land: Adjacency, bridges: Bridges): Adjacency {
    return {of: (tile) => [...land.of(tile), ...bridges.across(tile)]}
}
