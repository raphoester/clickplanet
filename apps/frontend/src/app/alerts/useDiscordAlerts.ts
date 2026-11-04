import {useEffect, useState} from "react"
import {DiscordAlerts} from "./discordAlerts.ts"

// What the game offers about its Discord server: to connect, while this account is not in it, or its page once it is.
// Neither: no Discord button, since the sign-in is the one way in with a flag's role.
export type DiscordOffer = {
    connect?: () => Promise<void>
    serverUrl?: string
}

const NOTHING: DiscordOffer = {}

// A Discord sign-in joins the server a second or two after it comes back: the backend's alerts module hears it, then
// asks Discord. The link is read again after each of these waits, until it reads linked.
export const JOIN_WAITS_MS: readonly number[] = [1_000, 2_000, 4_000]

// connectable: the server offers Discord's sign-in, which is the one way in.
// joining: this page came back from a Discord sign-in or link, which may not have joined yet.
export function useDiscordAlerts(
    alerts: DiscordAlerts | undefined, accountId: string | undefined, connectable: boolean, joining: boolean,
): DiscordOffer {
    const [offer, setOffer] = useState<DiscordOffer>(NOTHING)

    useEffect(() => {
        if (!alerts) return
        let stale = false
        let again: ReturnType<typeof setTimeout> | undefined
        const read = (waits: readonly number[]) => alerts.read().then(
            (link) => {
                if (stale) return
                if (!link.offered) setOffer(NOTHING)
                else if (link.linked) setOffer({serverUrl: link.serverUrl || undefined})
                else if (!connectable) setOffer(NOTHING)
                else {
                    setOffer({connect: alerts.connect})
                    alerts.joinIfAsked()
                }
                const [wait, ...later] = waits
                if (link.offered && !link.linked && wait !== undefined) again = setTimeout(() => void read(later), wait)
            },
            (e) => {
                console.error("Could not read the Discord link", e)
                if (!stale) setOffer(NOTHING)
            },
        )
        void read(joining ? JOIN_WAITS_MS : [])
        return () => {
            stale = true
            clearTimeout(again)
        }
    }, [alerts, accountId, connectable, joining])

    return offer
}
