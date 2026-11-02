import {DISCORD_INVITE, INSTAGRAM_PROFILE, TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../../links.ts"
import {CloseIcon, DiscordIcon, InstagramIcon, TikTokIcon, YouTubeIcon} from "../components/icons.tsx"
import {usePing} from "../alerts/usePing.ts"
import {useDiscordBar} from "./useDiscordBar.ts"

export type DiscordBarProps = {
    onPing?: () => Promise<void>
}

export default function DiscordBar({onPing}: DiscordBarProps) {
    const {hidden, hide} = useDiscordBar()
    if (hidden) return null

    return <div className="chat-discord">
        {onPing
            ? <PingButton leave={onPing}/>
            : <a href={DISCORD_INVITE}
                 target="_blank"
                 rel="noopener noreferrer"
                 className="chat-discord-link">
                <DiscordIcon size={24}/>
                <span className="chat-discord-label">Join Discord</span>
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

function PingButton({leave}: {leave: () => Promise<void>}) {
    const ping = usePing(leave)

    return <button type="button"
                   className="chat-discord-link chat-discord-ping"
                   aria-label={ping.name}
                   aria-busy={ping.busy}
                   disabled={ping.busy}
                   onClick={ping.press}>
        <DiscordIcon size={24}/>
        <span className="chat-discord-label">{ping.label}</span>
    </button>
}
