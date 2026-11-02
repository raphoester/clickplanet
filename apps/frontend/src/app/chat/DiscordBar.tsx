import {DISCORD_INVITE, INSTAGRAM_PROFILE, TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../../links.ts"
import {CloseIcon, DiscordIcon, InstagramIcon, TikTokIcon, YouTubeIcon} from "../components/icons.tsx"
import {pingFailureOf, pingHintOf, pingNameOf} from "../alerts/alertsMessages.ts"
import {DiscordOffer} from "../alerts/useDiscordAlerts.ts"
import {usePing} from "../alerts/usePing.ts"
import {useDiscordBar} from "./useDiscordBar.ts"

export type DiscordBarProps = {
    discord?: DiscordOffer
    // The flag the player plays for, by name: what the pings call it to defend.
    flagName?: string
}

export default function DiscordBar({discord, flagName}: DiscordBarProps) {
    const {hidden, hide} = useDiscordBar()
    if (hidden) return null

    return <div className={discord?.connect ? "chat-discord chat-discord--connect" : "chat-discord"}>
        {discord?.connect
            ? <ConnectButton connect={discord.connect} flagName={flagName}/>
            : <a href={discord?.serverUrl ?? DISCORD_INVITE}
                 target="_blank"
                 rel="noopener noreferrer"
                 className="chat-discord-link">
                <DiscordIcon size={24}/>
                <span className="chat-discord-label">{discord?.serverUrl ? "Open Discord" : "Join Discord"}</span>
            </a>}
        <a href={TIKTOK_PROFILE}
           target="_blank"
           rel="noopener noreferrer"
           className="chat-discord-other"
           aria-label="TikTok">
            <TikTokIcon size={20}/>
        </a>
        <a href={YOUTUBE_CHANNEL}
           target="_blank"
           rel="noopener noreferrer"
           className="chat-discord-other"
           aria-label="YouTube">
            <YouTubeIcon size={20}/>
        </a>
        <a href={INSTAGRAM_PROFILE}
           target="_blank"
           rel="noopener noreferrer"
           className="chat-discord-other"
           aria-label="Instagram">
            <InstagramIcon size={20}/>
        </a>
        <button type="button"
                className="chat-discord-close"
                aria-label="Hide the links"
                onClick={hide}>
            <CloseIcon size={16}/>
        </button>
    </div>
}

// Joins the server by Discord's sign-in, with the flag's role: the same "Join Discord", and why it is worth it on a
// line of its own under the bar, which the button's name already says.
function ConnectButton({connect, flagName}: {connect: () => Promise<void>, flagName?: string}) {
    const ping = usePing(connect)
    const label = ping.failure ? pingFailureOf(ping.failure) : "Join Discord"

    return <>
        <button type="button"
                className="chat-discord-link chat-discord-ping"
                aria-label={ping.failure ? label : pingNameOf(flagName)}
                aria-busy={ping.busy}
                disabled={ping.busy}
                onClick={ping.press}>
            <DiscordIcon size={24}/>
            <span className="chat-discord-label">{label}</span>
        </button>
        <p className="chat-discord-hint" aria-hidden="true">{pingHintOf(flagName)}</p>
    </>
}
