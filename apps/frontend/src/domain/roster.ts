import type {RosterEntry} from "../backends/player.ts"

export type RosterGroups = {
    /** Entries with a username. */
    players: RosterEntry[]
    guests: RosterEntry[]
}

/**
 * The roster split for the panel, each group in the order the server sent: it
 * has already sorted by name ignoring case, and a second sort here could only
 * disagree with it about what "ignoring case" means.
 */
export function rosterGroups(entries: readonly RosterEntry[]): RosterGroups {
    return {
        players: entries.filter((entry) => !entry.guest),
        guests: entries.filter((entry) => entry.guest),
    }
}
