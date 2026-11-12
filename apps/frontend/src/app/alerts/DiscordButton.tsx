import {ReactNode} from "react"
import {pingFailureOf} from "./alertsMessages.ts"
import {DiscordOffer} from "./useDiscordAlerts.ts"
import {usePing} from "./usePing.ts"
import {t} from "../../i18n/page.ts"

export type DiscordButtonProps = {
    offer?: DiscordOffer
    className: string
    children: ReactNode
}

// Discord's sign-in while the player is not in the server, the server's page once in, nothing otherwise.
export default function DiscordButton({offer, className, children}: DiscordButtonProps) {
    if (offer?.connect) return <JoinButton connect={offer.connect} className={className}>{children}</JoinButton>
    if (!offer?.serverUrl) return null

    return <a href={offer.serverUrl}
              target="_blank"
              rel="noopener noreferrer"
              className={className}
              aria-label={t("discord.open")}>
        {children}
    </a>
}

function JoinButton({connect, className, children}: {connect: () => Promise<void>, className: string, children: ReactNode}) {
    const ping = usePing(connect)

    return <button type="button"
                   className={className}
                   aria-label={ping.failure ? pingFailureOf(ping.failure) : t("discord.join")}
                   aria-busy={ping.busy}
                   disabled={ping.busy}
                   onClick={ping.press}>
        {children}
    </button>
}
