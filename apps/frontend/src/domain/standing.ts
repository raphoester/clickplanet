// The flags while the knock-out plays: each country with a capital, and the ones knocked out. Undefined while it does
// not play.
export type Standing = {
    capitals: ReadonlySet<string>
    out: ReadonlySet<string>
}

export function standingOf(capitals: ReadonlyMap<number, string>, out: ReadonlySet<string>): Standing {
    return {capitals: new Set(capitals.values()), out: new Set(out)}
}

// A flag still in: every flag while the knock-out does not play; while it does, a flag with a capital not knocked out.
export function stands(standing: Standing | undefined, flag: string): boolean {
    return standing === undefined || (standing.capitals.has(flag) && !standing.out.has(flag))
}

export function sameStanding(a: Standing | undefined, b: Standing | undefined): boolean {
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
