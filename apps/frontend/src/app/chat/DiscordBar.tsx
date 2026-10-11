import {DISCORD_INVITE} from "../../links.ts"
import {CloseIcon, DiscordIcon} from "../components/icons.tsx"
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
        <button type="button"
                className="chat-discord-close"
                aria-label="Hide the Discord link"
                onClick={hide}>
            <CloseIcon size={16}/>
        </button>
    </div>
}
