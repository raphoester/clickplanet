import type {RosterEntry, RosterEvent} from "../backends/player.ts"

export type RosterGroups = {
    players: RosterEntry[]
    guests: RosterEntry[]
}

export function applyRosterEvent(entries: readonly RosterEntry[], event: RosterEvent): readonly RosterEntry[] {
    switch (event.kind) {
        case "roster":
            return event.entries
        case "left":
            return entries.some((entry) => entry.key === event.key)
                ? entries.filter((entry) => entry.key !== event.key)
                : entries
        case "entry":
            return [...entries.filter((entry) => entry.key !== event.entry.key), event.entry].sort(compareRosterEntries)
    }
}

export function compareRosterEntries(a: RosterEntry, b: RosterEntry): number {
    return Number(a.guest) - Number(b.guest)
        || compareStrings(a.name.toLowerCase(), b.name.toLowerCase())
        || compareStrings(a.name, b.name)
        || compareStrings(a.key, b.key)
}

function compareStrings(a: string, b: string): number {
    return a < b ? -1 : a > b ? 1 : 0
}

export function rosterGroups(entries: readonly RosterEntry[]): RosterGroups {
    return {
        players: entries.filter((entry) => !entry.guest),
        guests: entries.filter((entry) => entry.guest),
    }
}
