import {Fragment, useCallback, useEffect, useLayoutEffect, useRef, useState} from "react";
import {ChatAnnouncement, ChatMessage, Reaction} from "../../backends/chat.ts";
import {PlayerLine} from "../../backends/player.ts";
import {Countries} from "../../domain/countries.ts";
import AdminCrown from "../components/AdminCrown.tsx";
import CountryFlag from "../components/CountryFlag.tsx";
import StreakFlame from "../components/StreakFlame.tsx";
import TitleBadge from "../titles/TitleBadge.tsx";
import {ChevronIcon} from "../components/icons.tsx";
import {ChatLogEntry, interleave, newestAt, startsGroup} from "../../domain/chatLog.ts";
import {describeBlast} from "../../domain/blast.ts";
import {truncate} from "../truncate.ts";
import {authorOf, authorStyle} from "./authorStyle.ts";
import ReactionBar, {AddReactionButton} from "./ReactionBar.tsx";

export type ChatLogProps = {
    messages: ChatMessage[]
    announcements?: ChatAnnouncement[]
    loading: boolean
    flashing?: ReadonlySet<string>
    onOpenPlayer?: (player: PlayerLine) => void
    onReact?: (messageId: string, reaction: Reaction, on: boolean) => void
    seenUntil?: number
    onSeen?: (at: number) => void
    watching?: boolean
}

const AUTHOR_MAX_LENGTH = 16

const clock = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})

const PINNED_SLACK_PX = 40

const SEEN_SLACK_PX = 4

const CONTEXT_ABOVE_UNSEEN_PX = 24

const NO_ANNOUNCEMENTS: ChatAnnouncement[] = []

export default function ChatLog(props: ChatLogProps) {
    const scroll = useRef<HTMLDivElement>(null)
    const pinned = useRef(true)
    const placed = useRef(false)
    const reported = useRef(0)
    const [behind, setBehind] = useState(false)
    const [picking, setPicking] = useState<string | undefined>(undefined)
    const announcements = props.announcements ?? NO_ANNOUNCEMENTS
    const listed = !props.loading && (props.messages.length > 0 || announcements.length > 0)
    const newest = newestAt(props.messages, announcements)

    // Decided once, as the log opens: it stays put while the player reads down from it.
    const [unseenAfter, setUnseenAfter] = useState<number | null>()
    if (unseenAfter === undefined && listed) {
        const seenUntil = props.seenUntil
        setUnseenAfter(seenUntil !== undefined && newest !== undefined && newest > seenUntil ? seenUntil : null)
    }

    const latest = useRef({onSeen: props.onSeen, watching: props.watching ?? true, seenUntil: props.seenUntil, newest})
    latest.current = {onSeen: props.onSeen, watching: props.watching ?? true, seenUntil: props.seenUntil, newest}

    const look = useCallback(() => {
        const element = scroll.current
        if (!element) return

        const {onSeen, watching, seenUntil, newest} = latest.current
        if (watching) {
            const shown = lastShownAt(element)
            if (shown !== undefined && shown > Math.max(seenUntil ?? 0, reported.current)) {
                reported.current = shown
                onSeen?.(shown)
            }
        }
        const seen = Math.max(seenUntil ?? 0, reported.current)
        setBehind(!pinned.current && newest !== undefined && newest > seen)
    }, [])

    useLayoutEffect(() => {
        const element = scroll.current
        if (!element) return

        if (!placed.current) {
            placed.current = true
            const unseen = element.querySelector<HTMLElement>(".chat-unseen")
            element.scrollTop = unseen
                ? Math.max(0, offsetIn(element, unseen) - CONTEXT_ABOVE_UNSEEN_PX)
                : element.scrollHeight
            pinned.current = distanceToBottom(element) <= PINNED_SLACK_PX
        } else if (pinned.current) {
            element.scrollTop = element.scrollHeight
        }
        look()
    }, [props.messages, announcements, unseenAfter, look])

    useEffect(() => {
        if (props.watching ?? true) look()
    }, [props.watching, look])

    useEffect(() => {
        const element = scroll.current
        if (!element || typeof ResizeObserver === "undefined") return

        const observer = new ResizeObserver(() => {
            if (pinned.current) element.scrollTop = element.scrollHeight
            look()
        })
        observer.observe(element)
        return () => observer.disconnect()
    }, [listed, look])

    const onScroll = () => {
        const element = scroll.current
        if (!element) return
        pinned.current = distanceToBottom(element) <= PINNED_SLACK_PX
        look()
    }

    const pick = (id: string) => (open: boolean) => setPicking(open ? id : undefined)

    const jumpToLatest = () => {
        const element = scroll.current
        if (!element) return
        element.scrollTop = element.scrollHeight
        pinned.current = true
        look()
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
                    const unseen = unseenAfter != null && atOf(entry) > unseenAfter
                        && (index === 0 || atOf(entries[index - 1]) <= unseenAfter)
                    return <Fragment key={entry.kind === "message" ? entry.message.id : entry.announcement.id}>
                        {unseen && <li className="chat-unseen" role="separator">Unread</li>}
                        {entry.kind === "announcement"
                            ? <AnnouncementLine announcement={entry.announcement}/>
                            : <MessageLine message={entry.message}
                                           previous={entries[index - 1]}
                                           flashing={props.flashing}
                                           picking={picking}
                                           pick={pick}
                                           onOpenPlayer={props.onOpenPlayer}
                                           onReact={props.onReact}/>}
                    </Fragment>
                })}
            </ul>
        </div>

        {behind && <button type="button" className="chat-jump" onClick={jumpToLatest}>
            New messages
            <ChevronIcon size={14}/>
        </button>}
    </div>
}

type MessageLineProps = {
    message: ChatMessage
    previous: ChatLogEntry | undefined
    flashing?: ReadonlySet<string>
    picking: string | undefined
    pick: (id: string) => (open: boolean) => void
    onOpenPlayer?: (player: PlayerLine) => void
    onReact?: (messageId: string, reaction: Reaction, on: boolean) => void
}

function MessageLine({message, previous, flashing, picking, pick, ...props}: MessageLineProps) {
    const author = authorOf(message)
    const opens = startsGroup(previous?.kind === "message" ? previous.message : undefined, message)

    return <li className={messageClass(flashing, message.id, opens)}
               style={authorStyle(author)}
               data-at={message.sentAt}>
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
                          onClick={() => props.onOpenPlayer?.(author)}>
                    {truncate(message.authorName, AUTHOR_MAX_LENGTH)}
                </button>
                : <span className="chat-message-author">
                    {truncate(message.authorName, AUTHOR_MAX_LENGTH)}
                </span>}
            {message.authorAdmin && <AdminCrown size={13}/>}
            <TitleBadge title={message.authorTitle}/>
            <StreakFlame days={message.authorStreak}/>
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
}

function AnnouncementLine({announcement}: {announcement: ChatAnnouncement}) {
    const bomber = countryName(announcement.country)
    const ground = announcement.ground === undefined ? undefined : Countries.get(announcement.ground)?.name

    return <li className="chat-announcement" data-at={announcement.announcedAt}>
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

function atOf(entry: ChatLogEntry): number {
    return entry.kind === "message" ? entry.message.sentAt : entry.announcement.announcedAt
}

function offsetIn(element: HTMLElement, child: HTMLElement): number {
    return child.getBoundingClientRect().top - element.getBoundingClientRect().top + element.scrollTop
}

function distanceToBottom(element: HTMLElement): number {
    return element.scrollHeight - element.scrollTop - element.clientHeight
}

// A line is seen once all of it has been on screen: the newest such line, or nothing.
function lastShownAt(element: HTMLElement): number | undefined {
    const bottom = element.getBoundingClientRect().bottom + SEEN_SLACK_PX
    const lines = element.querySelectorAll<HTMLElement>("[data-at]")
    for (let i = lines.length - 1; i >= 0; i--) {
        if (lines[i].getBoundingClientRect().bottom <= bottom) return Number(lines[i].dataset.at)
    }
    return undefined
}

function countryName(code: string): string {
    return Countries.get(code)?.name ?? code
}
