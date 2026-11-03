import {DISCORD_INVITE} from "../../links.ts"
import {CloseIcon, DiscordIcon} from "../components/icons.tsx"
import {pingFailureOf, pingHintOf, pingNameOf} from "../alerts/alertsMessages.ts"
import {DiscordOffer} from "../alerts/useDiscordAlerts.ts"
import {usePing} from "../alerts/usePing.ts"
import {useDiscordBar} from "./useDiscordBar.ts"

export type DiscordBarProps = {
    discord?: DiscordOffer
    // The flag the player plays for, by name: what the pings call it to defend.
    flagName?: string
}

// One call to action over the log: Discord, for the alert when the flag is under attack. The line says what it is
// for, the button under it does it, and the cross hides the box.
export default function DiscordBar({discord, flagName}: DiscordBarProps) {
    const {hidden, hide} = useDiscordBar()
    if (hidden) return null

    const connect = discord?.connect
    return <div className={connect ? "chat-discord chat-discord--connect panel-box" : "chat-discord panel-box"}>
        {connect && <p className="chat-discord-hint" aria-hidden="true">{pingHintOf(flagName)}</p>}
        {connect
            ? <ConnectButton connect={connect} flagName={flagName}/>
            : <a href={discord?.serverUrl ?? DISCORD_INVITE}
                 target="_blank"
                 rel="noopener noreferrer"
                 className="button button-mini chat-discord-join">
                <DiscordIcon size={20}/>
                <span className="chat-discord-label">{discord?.serverUrl ? "Open Discord" : "Join Discord"}</span>
            </a>}
        <button type="button"
                className="chat-discord-close"
                aria-label="Hide Discord"
                onClick={hide}>
            <CloseIcon size={16}/>
        </button>
    </div>
}

// Joins the server by Discord's sign-in, with the flag's role. The button's name says what for, which the line over
// it shows.
function ConnectButton({connect, flagName}: {connect: () => Promise<void>, flagName?: string}) {
    const ping = usePing(connect)
    const label = ping.failure ? pingFailureOf(ping.failure) : "Join Discord"

    return <button type="button"
                   className="button button-mini chat-discord-join"
                   aria-label={ping.failure ? label : pingNameOf(flagName)}
                   aria-busy={ping.busy}
                   disabled={ping.busy}
                   onClick={ping.press}>
        <DiscordIcon size={20}/>
        <span className="chat-discord-label">{label}</span>
    </button>
}
