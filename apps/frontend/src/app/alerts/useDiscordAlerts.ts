import {useEffect, useState} from "react"
import {DiscordAlerts} from "./discordAlerts.ts"

// Only while the server offers a link this account has not made.
export function useDiscordAlerts(alerts: DiscordAlerts | undefined, accountId: string | undefined): (() => Promise<void>) | undefined {
    const [offered, setOffered] = useState(false)

    useEffect(() => {
        if (!alerts) return
        let stale = false
        alerts.read().then(
            (link) => {
                if (!stale) setOffered(link.offered && !link.linked)
            },
            (e) => {
                console.error("Could not read the Discord link", e)
                if (!stale) setOffered(false)
            },
        )
        return () => {
            stale = true
        }
    }, [alerts, accountId])

    return offered ? alerts?.leave : undefined
}
