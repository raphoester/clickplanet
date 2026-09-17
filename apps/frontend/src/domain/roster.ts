import type {RosterEntry, RosterEvent} from "../backends/player.ts"

export type RosterGroups = {
    /** Entries with a username. */
    players: RosterEntry[]
    guests: RosterEntry[]
}

/**
 * The roster after one live event. A whole roster is taken as the server
 * ordered it; a single line goes where `compareRosterEntries` puts it. Answers
 * the list it was given when a player who left was not on it.
 */
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

/** The server's order: players before guests, then by name ignoring case, then by name, then by tag. */
export function compareRosterEntries(a: RosterEntry, b: RosterEntry): number {
    return Number(a.guest) - Number(b.guest)
        || compareStrings(a.name.toLowerCase(), b.name.toLowerCase())
        || compareStrings(a.name, b.name)
        || compareStrings(a.tag, b.tag)
}

function compareStrings(a: string, b: string): number {
    return a < b ? -1 : a > b ? 1 : 0
}

/** The roster split for the panel, each group in the roster's order. */
export function rosterGroups(entries: readonly RosterEntry[]): RosterGroups {
    return {
        players: entries.filter((entry) => !entry.guest),
        guests: entries.filter((entry) => entry.guest),
    }
}
