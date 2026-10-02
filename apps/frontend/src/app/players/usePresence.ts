import {useEffect, useRef} from "react"
import {PresenceBackend} from "../../backends/player.ts"
import {Announcing, PresenceSchedule, SETTLE_MS} from "../../domain/presence.ts"

const CHECK_MS = SETTLE_MS

export function usePresence(backend: PresenceBackend | undefined, announcing: Announcing): void {
    const schedule = useRef<PresenceSchedule>()
    schedule.current ??= new PresenceSchedule(announcing)

    const {countryCode, username} = announcing
    useEffect(() => {
        schedule.current!.want({countryCode, username}, Date.now())
    }, [countryCode, username])

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

    useEffect(() => {
        if (!backend) return

        const onPageHide = (event: PageTransitionEvent) => {
            if (!event.persisted) backend.leave()
        }
        window.addEventListener("pagehide", onPageHide)
        return () => window.removeEventListener("pagehide", onPageHide)
    }, [backend])
}
