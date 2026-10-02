import {useEffect, useState} from "react"
import {PresenceBackend, RosterEntry} from "../../backends/player.ts"
import {applyRosterEvent} from "../../domain/roster.ts"

export type RosterState =
    | {kind: "loading"}
    | {kind: "unavailable"}
    | {kind: "ready", entries: readonly RosterEntry[]}

const UNAVAILABLE: RosterState = {kind: "unavailable"}

export function useRoster(backend: PresenceBackend | undefined): RosterState {
    const [state, setState] = useState<RosterState>(() => backend ? {kind: "loading"} : UNAVAILABLE)

    useEffect(() => {
        if (!backend) {
            setState(UNAVAILABLE)
            return
        }

        setState({kind: "loading"})
        return backend.listenForRoster(
            (event) => setState((current) => {
                if (current.kind !== "ready") {
                    return event.kind === "roster" ? {kind: "ready", entries: event.entries} : current
                }
                const entries = applyRosterEvent(current.entries, event)
                return entries === current.entries ? current : {kind: "ready", entries}
            }),
            () => setState(UNAVAILABLE),
        )
    }, [backend])

    return state
}
