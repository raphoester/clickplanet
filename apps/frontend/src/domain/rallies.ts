// A ring a player called on the globe for its flag: until endsAt, a click of that flag on a tile inside it takes cost
// from the bank instead of a whole click. Every flag sees it.
export type Rally = {
    id: string
    accountId: string
    countryId: string
    // The ring's centre.
    tile: number
    // In radians of arc on the unit globe: a tile whose centre is that close to the centre's or closer is inside.
    radius: number
    // On the page's own clock (performance.now()), from the time the server said it had left.
    endsAt: number
}

export const WHOLE_CLICK = 1

// The rallies still on, the newest last: one told again replaces the one before, by id.
export class Rallies {
    private readonly byId = new Map<string, Rally>()

    public keep(rally: Rally): void {
        this.byId.delete(rally.id)
        this.byId.set(rally.id, rally)
    }

    // Every rally the server answered: the ones it no longer has are over.
    public replace(rallies: readonly Rally[]): void {
        this.byId.clear()
        for (const rally of rallies) this.keep(rally)
    }

    // Forgets the ones over, and answers whether any was.
    public prune(now: number): boolean {
        let pruned = false
        for (const [id, rally] of this.byId) {
            if (rally.endsAt > now) continue
            this.byId.delete(id)
            pruned = true
        }
        return pruned
    }

    public on(now: number): Rally[] {
        return [...this.byId.values()].filter((rally) => rally.endsAt > now)
    }
}

export function covers(rally: Rally, tile: number, positions: ArrayLike<number>): boolean {
    const a = (rally.tile - 1) * 3
    const b = (tile - 1) * 3
    if (rally.tile < 1 || tile < 1 || b + 2 >= positions.length || a + 2 >= positions.length) return false

    const along = positions[a] * positions[b] + positions[a + 1] * positions[b + 1] + positions[a + 2] * positions[b + 2]
    const lengths = Math.hypot(positions[a], positions[a + 1], positions[a + 2]) * Math.hypot(positions[b], positions[b + 1], positions[b + 2])
    return lengths > 0 && along / lengths >= Math.cos(rally.radius)
}

// What a click of the flag on the tile takes from the bank: the rally's cost inside a ring of the flag among the rallies
// on, a whole click elsewhere. Rings that overlap cost no less. The server decides; this is what the page counts until it
// answers.
export function clickCostOf(
    tile: number, flag: string, on: readonly Rally[], positions: ArrayLike<number>, cost: number | undefined,
): number {
    if (cost === undefined || cost <= 0 || cost >= WHOLE_CLICK) return WHOLE_CLICK
    const inside = on.some((rally) => rally.countryId === flag && covers(rally, tile, positions))
    return inside ? cost : WHOLE_CLICK
}
