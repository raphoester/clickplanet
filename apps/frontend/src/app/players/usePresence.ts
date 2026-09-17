import {useEffect, useRef} from "react"
import {PresenceBackend} from "../../backends/player.ts"
import {Announcing, PresenceSchedule, SETTLE_MS} from "../../domain/presence.ts"

/**
 * How often the schedule is asked. Asking costs a comparison: the held token is
 * read from memory, never minted. It is this short so a token a click just
 * minted, or a change that just settled, is announced within about a second —
 * there is no event to wait on for either.
 */
const CHECK_MS = SETTLE_MS

/**
 * Tells the server this player is playing, under its flag and name, while the
 * page is open. The rules are `PresenceSchedule`'s; this only runs its clock.
 */
export function usePresence(backend: PresenceBackend | undefined, announcing: Announcing): void {
    const schedule = useRef<PresenceSchedule>()
    schedule.current ??= new PresenceSchedule(announcing)

    const {countryCode, guestName, username} = announcing
    useEffect(() => {
        schedule.current!.want({countryCode, guestName, username}, Date.now())
    }, [countryCode, guestName, username])

    useEffect(() => {
        if (!backend) return
        const current = schedule.current!

        const check = () => {
            const presence = current.claim(Date.now(), backend.heldSession())
            if (!presence) return

            backend.announce(presence)
                .catch((e) => console.error("Announce failed", e))
                .finally(() => current.settle())
        }

        check()
        const timer = window.setInterval(check, CHECK_MS)
        return () => window.clearInterval(timer)
    }, [backend])
}
