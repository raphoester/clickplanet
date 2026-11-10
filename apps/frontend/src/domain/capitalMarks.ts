import type {FrontChanges} from "./front.ts"

// The capitals still standing: each one held by its own country.
export class CapitalMarks {
    private readonly marked = new Set<number>()

    constructor(
        private readonly capitals: ReadonlyMap<number, string>,
        private readonly ownerOf: (tile: number) => string | undefined,
    ) {}

    follow(): FrontChanges {
        const off = [...this.marked]
        this.marked.clear()
        for (const tile of this.capitals.keys()) {
            if (this.stands(tile)) this.marked.add(tile)
        }
        const on = [...this.marked]
        return {on, off: off.filter((tile) => !this.marked.has(tile))}
    }

    touch(tiles: readonly number[]): FrontChanges {
        const changes: FrontChanges = {on: [], off: []}
        for (const tile of tiles) {
            if (!this.capitals.has(tile)) continue
            const stands = this.stands(tile)
            if (stands === this.marked.has(tile)) continue
            if (stands) {
                this.marked.add(tile)
                changes.on.push(tile)
            } else {
                this.marked.delete(tile)
                changes.off.push(tile)
            }
        }
        return changes
    }

    drop(): FrontChanges {
        const off = [...this.marked]
        this.marked.clear()
        return {on: [], off}
    }

    private stands(tile: number): boolean {
        return this.ownerOf(tile) === this.capitals.get(tile)
    }
}
