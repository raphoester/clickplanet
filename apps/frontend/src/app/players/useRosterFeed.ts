import {useEffect} from "react"
import {PresenceBackend} from "../../backends/player.ts"
import {useRoster} from "./rosterStore.ts"

export function useRosterFeed(backend: PresenceBackend | undefined): void {
    useEffect(() => {
        if (!backend) return
        const {follow, hear, lose} = useRoster.getState()
        follow()
        const stop = backend.listenForRoster(hear, lose)
        // Nobody is listed once nothing follows the stream.
        return () => {
            stop()
            lose()
        }
    }, [backend])
}
