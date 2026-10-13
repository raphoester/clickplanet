import {DISCORD_INVITE, INSTAGRAM_PROFILE, TIKTOK_PROFILE} from "../../links.ts"
import {CloseIcon, DiscordIcon, InstagramIcon, TikTokIcon} from "../components/icons.tsx"
import {useDiscordBar} from "./useDiscordBar.ts"

export default function DiscordBar() {
    const {hidden, hide} = useDiscordBar()
    if (hidden) return null

    return <div className="chat-discord">
        <a href={DISCORD_INVITE}
           target="_blank"
           rel="noopener noreferrer"
           className="chat-discord-link">
            <DiscordIcon size={24}/>
            <span className="chat-discord-label">Join the Discord</span>
        </a>
        <a href={TIKTOK_PROFILE}
           target="_blank"
           rel="noopener noreferrer"
           className="chat-discord-other"
           aria-label="TikTok">
            <TikTokIcon size={20}/>
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
