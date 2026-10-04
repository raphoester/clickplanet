import {useEffect, useRef, useState} from "react"
import {PlayerTitle, PresenceBackend, RosterEntry} from "../../backends/player.ts"
import {SETTLE_MS} from "../../domain/presence.ts"
import {applyRosterEvent} from "../../domain/roster.ts"

export type RosterState =
    | {kind: "loading"}
    | {kind: "unavailable"}
    | {kind: "ready", entries: readonly RosterEntry[]}

const UNAVAILABLE: RosterState = {kind: "unavailable"}

export function useRoster(backend: PresenceBackend | undefined, onTitleEarned?: (title: PlayerTitle) => void): RosterState {
    const [state, setState] = useState<RosterState>(() => backend ? {kind: "loading"} : UNAVAILABLE)
    const [session, setSession] = useState(() => backend?.heldIdentity())
    const titleEarned = useRef(onTitleEarned)
    titleEarned.current = onTitleEarned

    useEffect(() => {
        if (!backend) return
        const check = () => setSession(backend.heldIdentity())
        const timer = window.setInterval(check, SETTLE_MS)
        return () => window.clearInterval(timer)
    }, [backend])

    useEffect(() => {
        if (!backend) {
            setState(UNAVAILABLE)
            return
        }

        setState((current) => current.kind === "ready" ? current : {kind: "loading"})
        return backend.listenForRoster(
            (event) => setState((current) => {
                if (current.kind !== "ready") {
                    return event.kind === "roster" ? {kind: "ready", entries: event.entries} : current
                }
                const entries = applyRosterEvent(current.entries, event)
                return entries === current.entries ? current : {kind: "ready", entries}
            }),
            () => setState(UNAVAILABLE),
            (title) => titleEarned.current?.(title),
        )
    }, [backend, session])

    return state
}
