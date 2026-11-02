import {useEffect, useState} from "react"
import {DiscordAlerts} from "./discordAlerts.ts"

// What the game offers about its Discord server: to connect, while this account is not in it, or its page once it is.
// Neither: the invite alone.
export type DiscordOffer = {
    connect?: () => Promise<void>
    serverUrl?: string
}

const NOTHING: DiscordOffer = {}

// connectable: the server offers Discord's sign-in, which is the one way in.
export function useDiscordAlerts(alerts: DiscordAlerts | undefined, accountId: string | undefined, connectable: boolean): DiscordOffer {
    const [offer, setOffer] = useState<DiscordOffer>(NOTHING)

    useEffect(() => {
        if (!alerts) return
        let stale = false
        alerts.read().then(
            (link) => {
                if (stale) return
                if (!link.offered) setOffer(NOTHING)
                else if (link.linked) setOffer({serverUrl: link.serverUrl || undefined})
                else setOffer(connectable ? {connect: alerts.connect} : NOTHING)
            },
            (e) => {
                console.error("Could not read the Discord link", e)
                if (!stale) setOffer(NOTHING)
            },
        )
        return () => {
            stale = true
        }
    }, [alerts, accountId, connectable])

    return offer
}
