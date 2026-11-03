export type GhostTake = {tile: number, flag: string, until: number}

type OwnerOf = (tile: number) => string | undefined

// The tiles ghosts took on this browser: painted over the server's flag, never sent and never counted by the server.
export class Ghosts {
    private readonly takes = new Map<number, GhostTake>()

    constructor(saved: readonly GhostTake[], now: number) {
        for (const take of saved) {
            if (take.until > now) this.takes.set(take.tile, take)
        }
    }

    public get size(): number {
        return this.takes.size
    }

    public flagOf(tile: number): string | undefined {
        return this.takes.get(tile)?.flag
    }

    public take(take: GhostTake): void {
        this.takes.set(take.tile, take)
    }

    public release(tiles: Iterable<number>): number[] {
        const freed: number[] = []
        if (this.takes.size === 0) return freed
        for (const tile of tiles) {
            if (this.takes.delete(tile)) freed.push(tile)
        }
        return freed
    }

    public expire(now: number): number[] {
        const freed: number[] = []
        for (const {tile, until} of this.takes.values()) {
            if (until <= now) freed.push(tile)
        }
        for (const tile of freed) this.takes.delete(tile)
        return freed
    }

    public countsOver(counts: ReadonlyMap<string, number>, ownerOf: OwnerOf): ReadonlyMap<string, number> {
        if (this.takes.size === 0) return counts
        const seen = new Map(counts)
        for (const {tile, flag} of this.takes.values()) {
            seen.set(flag, (seen.get(flag) ?? 0) + 1)
            const holder = ownerOf(tile)
            if (holder !== undefined) seen.set(holder, Math.max(0, (seen.get(holder) ?? 0) - 1))
        }
        return seen
    }

    public heldFrom(flag: string, ownerOf: OwnerOf): number {
        let held = 0
        for (const {tile} of this.takes.values()) {
            if (ownerOf(tile) === flag) held++
        }
        return held
    }

    public snapshot(): GhostTake[] {
        return [...this.takes.values()]
    }
}

export function ghostTakesOf(saved: unknown, size: number): GhostTake[] {
    if (!Array.isArray(saved)) return []
    return saved.filter((take): take is GhostTake =>
        typeof take === "object" && take !== null
        && Number.isInteger(take.tile) && take.tile >= 1 && take.tile <= size
        && typeof take.flag === "string" && take.flag !== ""
        && typeof take.until === "number" && Number.isFinite(take.until))
}
