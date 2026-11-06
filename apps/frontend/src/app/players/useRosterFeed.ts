import {useEffect} from "react"
import {PresenceBackend, RosterEntry, RosterEvent} from "../../backends/player.ts"
import {useNameplates} from "../nameplates/nameplatesStore.ts"
import {useRoster} from "./rosterStore.ts"

export function useRosterFeed(backend: PresenceBackend): void {
    useEffect(() => {
        const {follow, hear, lose} = useRoster.getState()
        follow()
        const stop = backend.listenForRoster((event) => {
            hear(event)
            // Who is online is named at once: their lines are as they are now.
            useNameplates.getState().learn(linesOf(event))
        }, lose)
        // Nobody is listed once nothing follows the stream.
        return () => {
            stop()
            lose()
        }
    }, [backend])
}

function linesOf(event: RosterEvent): readonly RosterEntry[] {
    switch (event.kind) {
        case "roster":
            return event.entries
        case "entry":
            return [event.entry]
        case "left":
            return []
    }
}
