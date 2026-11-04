import {create} from "zustand"
import {RosterEntry, RosterEvent} from "../../backends/player.ts"
import {applyRosterEvent} from "../../domain/roster.ts"

export type RosterState =
    | {kind: "loading"}
    | {kind: "unavailable"}
    | {kind: "ready", entries: readonly RosterEntry[]}

type RosterStore = {
    state: RosterState
    follow: () => void
    hear: (event: RosterEvent) => void
    lose: () => void
}

const UNAVAILABLE: RosterState = {kind: "unavailable"}

const NOBODY: readonly RosterEntry[] = []

export const onlineCount = (roster: RosterStore) => roster.state.kind === "ready" ? roster.state.entries.length : undefined

export const rosterEntries = (roster: RosterStore) => roster.state.kind === "ready" ? roster.state.entries : NOBODY

export const useRoster = create<RosterStore>()((set) => ({
    state: UNAVAILABLE,
    follow: () => set({state: {kind: "loading"}}),
    hear: (event) => set((roster) => {
        const {state} = roster
        if (state.kind !== "ready") {
            return event.kind === "roster" ? {state: {kind: "ready", entries: event.entries}} : roster
        }
        const entries = applyRosterEvent(state.entries, event)
        return entries === state.entries ? roster : {state: {kind: "ready", entries}}
    }),
    lose: () => set({state: UNAVAILABLE}),
}))
