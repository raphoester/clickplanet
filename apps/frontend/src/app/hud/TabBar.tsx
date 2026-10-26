import {ReactNode} from "react"
import {ChatIcon, SettingsIcon, ShareIcon, TargetIcon, TrophyIcon, UserIcon} from "../components/icons.tsx"
import "./TabBar.css"

export type Tab = "board" | "missions" | "chat" | "you" | "share" | "settings"

const UNREAD_CAP = 99

export type TabBarProps = {
    open?: string
    onOpen: (tab: Tab) => void
    chat: boolean
    unread: number
    you?: "guest" | "player"
    missions?: number
}

export default function TabBar({open, onOpen, chat, unread, you, missions}: TabBarProps) {
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
        {missions !== undefined && tab("missions", "Missions", <TargetIcon/>,
            missions > 0 && <span className="chip tab-bar-badge" key={missions} aria-hidden="true">{missions}</span>,
            missions === 0 ? undefined : missions === 1 ? "Missions, 1 new" : `Missions, ${missions} new`)}
        {chat && tab("chat", "Chat", <ChatIcon/>,
            unread > 0 && <span className="chip tab-bar-badge" key={unread} aria-hidden="true">
                {unread > UNREAD_CAP ? `${UNREAD_CAP}+` : unread}
            </span>,
            unread === 0 ? undefined : unread === 1 ? "Chat, 1 new message" : `Chat, ${unread} new messages`)}
        {you && tab("you", you === "guest" ? "Sign in" : "You", <UserIcon/>,
            you === "guest" && <span className="tab-bar-dot" aria-hidden="true"/>)}
        {tab("share", "Share", <ShareIcon size={22}/>)}
        {tab("settings", "Settings", <SettingsIcon/>)}
    </nav>
}
