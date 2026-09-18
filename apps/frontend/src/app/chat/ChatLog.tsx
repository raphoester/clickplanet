import {useEffect, useRef, useState} from "react";
import {ChatAnnouncement, ChatMessage, GUEST_PREFIX, Reaction} from "../../backends/chat.ts";
import {PlayerLine} from "../../backends/player.ts";
import {Countries} from "../../domain/countries.ts";
import AdminCrown from "../components/AdminCrown.tsx";
import CountryFlag from "../components/CountryFlag.tsx";
import {ChevronIcon} from "../components/icons.tsx";
import {interleave, startsGroup} from "../../domain/chatLog.ts";
import {describeBlast} from "../../domain/blast.ts";
import {truncate} from "../truncate.ts";
import {authorStyle} from "./authorStyle.ts";
import ReactionBar, {AddReactionButton} from "./ReactionBar.tsx";

export type ChatLogProps = {
    messages: ChatMessage[]
    /** Lines nobody sent, shown between the messages by time. */
    announcements?: ChatAnnouncement[]
    loading: boolean
    flashing?: ReadonlySet<string>
    /** Absent, an author's name is plain text. */
    onOpenPlayer?: (player: PlayerLine) => void
    /** Absent, reactions are shown and none can be given. */
    onReact?: (messageId: string, reaction: Reaction, on: boolean) => void
}

const AUTHOR_MAX_LENGTH = 16

const clock = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})

const PINNED_SLACK_PX = 40

const NO_ANNOUNCEMENTS: ChatAnnouncement[] = []

export default function ChatLog(props: ChatLogProps) {
    const scroll = useRef<HTMLDivElement>(null)
    const pinned = useRef(true)
    const lastId = useRef<string | undefined>(undefined)
    const [behind, setBehind] = useState(false)
    const [picking, setPicking] = useState<string | undefined>(undefined)
    const announcements = props.announcements ?? NO_ANNOUNCEMENTS

    useEffect(() => {
        const element = scroll.current
        const last = props.messages[props.messages.length - 1]?.id
        const arrived = last !== lastId.current
        lastId.current = last
        if (!element) return

        // Reading a message further up is not interrupted by a new one landing:
        // the pill says it is there instead of yanking the log down. A reaction
        // is not a new message, and says nothing.
        if (!pinned.current) {
            if (arrived) setBehind(true)
            return
        }

        element.scrollTop = element.scrollHeight
    }, [props.messages, announcements])

    const onScroll = () => {
        const element = scroll.current
        if (!element) return
        const distance = element.scrollHeight - element.scrollTop - element.clientHeight
        pinned.current = distance <= PINNED_SLACK_PX
        if (pinned.current) setBehind(false)
    }

    const pick = (id: string) => (open: boolean) => setPicking(open ? id : undefined)

    const jumpToLatest = () => {
        const element = scroll.current
        if (!element) return
        element.scrollTop = element.scrollHeight
        pinned.current = true
        setBehind(false)
    }

    if (props.loading) {
        return <div className="chat-log chat-log-empty">
            <p>Loading the chat…</p>
        </div>
    }

    if (props.messages.length === 0 && announcements.length === 0) {
        return <div className="chat-log chat-log-empty">
            <p>Nobody has said anything yet. Go on.</p>
        </div>
    }

    return <div className="chat-log-shell">
        <div className="chat-log" ref={scroll} onScroll={onScroll}>
            <ul className="chat-messages" aria-live="polite">
                {interleave(props.messages, announcements).map((entry, index, entries) => {
                    if (entry.kind === "announcement") {
                        return <AnnouncementLine key={entry.announcement.id} announcement={entry.announcement}/>
                    }

                    // A line between two messages ends the run above it: the next one says who is talking again.
                    const message = entry.message
                    const previous = entries[index - 1]
                    const opens = startsGroup(previous?.kind === "message" ? previous.message : undefined, message)

                    return <li key={message.id}
                               className={messageClass(props.flashing, message.id, opens)}
                               style={authorStyle(message.authorName, message.authorTag)}>
                        {opens && <div className="chat-message-head">
                            <span className="chat-message-country"
                                  role="img"
                                  aria-label={countryName(message.countryCode)}
                                  title={countryName(message.countryCode)}>
                                <CountryFlag code={message.countryCode}/>
                            </span>
                            {props.onOpenPlayer
                                ? <button type="button"
                                          className="chat-message-author player-name-button"
                                          title={message.authorName}
                                          onClick={() => props.onOpenPlayer?.(authorOf(message))}>
                                    {truncate(message.authorName, AUTHOR_MAX_LENGTH)}
                                </button>
                                : <span className="chat-message-author">
                                    {truncate(message.authorName, AUTHOR_MAX_LENGTH)}
                                </span>}
                            {message.authorAdmin && <AdminCrown size={13}/>}
                            <span className="chat-message-tag">#{message.authorTag}</span>
                            <time className="chat-message-time"
                                  dateTime={new Date(message.sentAt).toISOString()}>
                                {clock.format(message.sentAt)}
                            </time>
                        </div>}

                        <div className="chat-message-line">
                            <p className="chat-message-text">{message.text}</p>
                            {props.onReact && <AddReactionButton messageId={message.id}
                                                                 picking={picking === message.id}
                                                                 setPicking={pick(message.id)}/>}
                        </div>

                        <ReactionBar messageId={message.id}
                                     reactions={message.reactions}
                                     onReact={props.onReact && ((reaction, on) => props.onReact?.(message.id, reaction, on))}
                                     picking={picking === message.id}
                                     setPicking={pick(message.id)}/>
                    </li>
                })}
            </ul>
        </div>

        {behind && <button type="button" className="chat-jump" onClick={jumpToLatest}>
            New messages
            <ChevronIcon size={14}/>
        </button>}
    </div>
}

/**
 * A line the chat says on its own: no bubble, no author, no reactions. It reads
 * like the news line at the top of the screen, and stays.
 */
function AnnouncementLine({announcement}: {announcement: ChatAnnouncement}) {
    const bomber = countryName(announcement.country)
    const ground = announcement.ground === undefined ? undefined : Countries.get(announcement.ground)?.name

    return <li className="chat-announcement">
        <span className="chat-announcement-icon" aria-hidden="true">
            {announcement.tile === undefined ? "🌊" : "💥"}
        </span>
        <CountryFlag code={announcement.country}/>
        <span className="chat-announcement-text">
            <strong>{bomber}</strong> {describeBlast(announcement, ground)}
        </span>
        <time className="chat-announcement-time" dateTime={new Date(announcement.announcedAt).toISOString()}>
            {clock.format(announcement.announcedAt)}
        </time>
    </li>
}

// No username starts with the prefix, so the name alone says who is a guest.
function authorOf(message: ChatMessage): PlayerLine {
    return {
        name: message.authorName,
        tag: message.authorTag,
        countryCode: message.countryCode,
        guest: message.authorName.startsWith(GUEST_PREFIX),
        admin: message.authorAdmin,
    }
}

function messageClass(
    flashing: ReadonlySet<string> | undefined,
    id: string,
    opens: boolean,
): string {
    return [
        "chat-message",
        opens ? "chat-message-opens" : "chat-message-cont",
        flashing?.has(id) && "chat-message-new",
    ].filter(Boolean).join(" ")
}

// The badge is a flag and nothing else, so the name it stands for is what the
// tooltip and the accessibility tree carry.
function countryName(code: string): string {
    return Countries.get(code)?.name ?? code
}
