// An arena's flags: each country with a capital, and the ones the arena knocked out. Outside an arena, undefined.
export type ArenaFlags = {
    capitals: ReadonlySet<string>
    out: ReadonlySet<string>
}

export function arenaFlagsOf(capitals: ReadonlyMap<number, string>, out: ReadonlySet<string>): ArenaFlags {
    return {capitals: new Set(capitals.values()), out: new Set(out)}
}

// A flag still in: outside an arena every flag, in one a flag with a capital not knocked out.
export function stands(arena: ArenaFlags | undefined, flag: string): boolean {
    return arena === undefined || (arena.capitals.has(flag) && !arena.out.has(flag))
}

export function sameArenaFlags(a: ArenaFlags | undefined, b: ArenaFlags | undefined): boolean {
    if (a === undefined || b === undefined) return a === b
    return sameSet(a.capitals, b.capitals) && sameSet(a.out, b.out)
}

function sameSet(a: ReadonlySet<string>, b: ReadonlySet<string>): boolean {
    if (a.size !== b.size) return false
    for (const flag of a) {
        if (!b.has(flag)) return false
    }
    return true
}
