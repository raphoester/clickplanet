const NONE: readonly number[] = []

// Each body of water the lattice draws, by its coast tiles; a tile between two is on the coast of both.
export class Waters {
    private readonly bodies: readonly (readonly number[])[]
    private readonly onTile = new Map<number, number[]>()

    constructor(bodies: readonly (readonly number[])[]) {
        this.bodies = bodies
        bodies.forEach((coast, body) => {
            for (const tile of coast) {
                const touched = this.onTile.get(tile)
                if (touched) touched.push(body)
                else this.onTile.set(tile, [body])
            }
        })
    }

    bodiesOf(tile: number): readonly number[] {
        return this.onTile.get(tile) ?? NONE
    }

    coastOf(body: number): readonly number[] {
        return this.bodies[body] ?? NONE
    }
}
