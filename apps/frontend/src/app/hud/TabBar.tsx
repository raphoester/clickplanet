import {ReactNode} from "react"
import {ChatIcon, MoreIcon, TrophyIcon, UserIcon} from "../components/icons.tsx"
import "./TabBar.css"

export type Tab = "board" | "chat" | "you" | "more"

const UNREAD_CAP = 99

export type TabBarProps = {
    open?: string
    onOpen: (tab: Tab) => void
    chat: boolean
    unread: number
    you?: "guest" | "player"
}

export default function TabBar({open, onOpen, chat, unread, you}: TabBarProps) {
    const tab = (id: Tab, label: string, icon: ReactNode, extra?: ReactNode, name?: string) =>
        <button type="button"
                className={open === id ? "tab-bar-tab tab-bar-tab--open" : "tab-bar-tab"}
                aria-expanded={open === id}
                aria-label={name}
                onClick={() => onOpen(id)}>
            {icon}
            <span className="tab-bar-label">{label}</span>
            {extra}
        </button>

    return <nav className="tab-bar panel" aria-label="Game">
        {tab("board", "Board", <TrophyIcon/>)}
        {chat && tab("chat", "Chat", <ChatIcon/>,
            unread > 0 && <span className="chip tab-bar-badge" key={unread} aria-hidden="true">
                {unread > UNREAD_CAP ? `${UNREAD_CAP}+` : unread}
            </span>,
            unread === 0 ? undefined : unread === 1 ? "Chat, 1 new message" : `Chat, ${unread} new messages`)}
        {you && tab("you", you === "guest" ? "Sign in" : "You", <UserIcon/>,
            you === "guest" && <span className="tab-bar-dot" aria-hidden="true"/>)}
        {tab("more", "More", <MoreIcon/>)}
    </nav>
}
