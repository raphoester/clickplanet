import {useEffect, useState} from "react"
import {PresenceBackend, RosterEntry, RosterUnavailableError} from "../../backends/player.ts"

/** A proxy may serve the roster for 5s, so twice that is as fresh as asking gets. */
export const ROSTER_EVERY_MS = 10_000

export type RosterState =
    | {kind: "loading"}
    /** No backend wired, or a server without the roster: the menu offers no list. */
    | {kind: "unavailable"}
    | {kind: "ready", entries: RosterEntry[]}

const UNAVAILABLE: RosterState = {kind: "unavailable"}

/**
 * The roster, read at once, then every `ROSTER_EVERY_MS` while the tab is
 * visible, and at once again when it becomes visible. A hidden tab asks
 * nothing: nobody is reading the list, and a background tab left open all day
 * would otherwise keep a request going every ten seconds.
 *
 * A read that fails keeps the last list — one dropped poll is not worth an
 * empty panel. A server without the roster hides it for good.
 */
export function useRoster(backend: PresenceBackend | undefined): RosterState {
    const [state, setState] = useState<RosterState>(() => backend ? {kind: "loading"} : UNAVAILABLE)

    useEffect(() => {
        if (!backend) {
            setState(UNAVAILABLE)
            return
        }

        let stopped = false
        let reading = false

        const read = async () => {
            if (stopped || reading) return
            reading = true
            try {
                const entries = await backend.roster()
                if (!stopped) setState({kind: "ready", entries})
            } catch (e) {
                if (stopped) return
                if (e instanceof RosterUnavailableError) {
                    stop()
                    setState(UNAVAILABLE)
                    return
                }
                console.error("Could not read the roster", e)
            } finally {
                reading = false
            }
        }

        const onVisibilityChange = () => {
            if (document.visibilityState === "visible") void read()
        }

        const timer = window.setInterval(onVisibilityChange, ROSTER_EVERY_MS)
        document.addEventListener("visibilitychange", onVisibilityChange)

        const stop = () => {
            stopped = true
            window.clearInterval(timer)
            document.removeEventListener("visibilitychange", onVisibilityChange)
        }

        void read()
        return stop
    }, [backend])

    return state
}
