import {useEffect, useRef, useState} from "react"
import {PlayerTitle, PresenceBackend} from "../../backends/player.ts"
import {SETTLE_MS} from "../../domain/presence.ts"
import {useRoster} from "./rosterStore.ts"

// Follows the roster stream into the store, and hands on the titles the same stream brings.
export function useRosterFeed(backend: PresenceBackend | undefined, onTitleEarned?: (title: PlayerTitle) => void): void {
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
        if (!backend) return
        const {follow, hear, lose} = useRoster.getState()
        follow()
        return backend.listenForRoster(hear, lose, (title) => titleEarned.current?.(title))
    }, [backend, session])

    // Nobody is listed once nothing follows the stream.
    useEffect(() => {
        if (!backend) return
        return () => useRoster.getState().lose()
    }, [backend])
}
