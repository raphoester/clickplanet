import {useCallback, useEffect, useId, useRef, useState} from "react";
import {ChatBackend, OutgoingMessage} from "../../backends/chat.ts";
import {Country} from "../../domain/countries.ts";
import {idsSince, unreadSince} from "../../domain/chatLog.ts";
import {ChevronIcon} from "../components/icons.tsx";
import {opensFolded} from "../compact.ts";
import {truncate} from "../truncate.ts";
import {PlaySound} from "../sound/soundPlayer.ts";
import {authorStyle} from "./authorStyle.ts";
import ChatComposer from "./ChatComposer.tsx";
import ChatLog from "./ChatLog.tsx";
import {useChat} from "./useChat.ts";
import {useChatIdentity} from "./useChatIdentity.ts";
import "./ChatPanel.css"

export type ChatPanelProps = {
    backend?: ChatBackend
    country: Country
    playSound?: PlaySound
}

const UNREAD_CAP = 99

const PEEK_AUTHOR_MAX_LENGTH = 12

/** How long a message stays lit after it lands. Matches `chat-message-glow`. */
const FLASH_MS = 1600

const NOTHING: ReadonlySet<string> = new Set()

export default function ChatPanel(props: ChatPanelProps) {
    const [isOpen, setIsOpen] = useState(() => !opensFolded())
    const [unread, setUnread] = useState(0)
    const [flashing, setFlashing] = useState<ReadonlySet<string>>(NOTHING)
    const bodyId = useId()

    const {messages, mine, status, failure, send} = useChat({backend: props.backend})
    const {identity, setName} = useChatIdentity()

    const lastSeen = useRef<string | undefined>(undefined)
    const seenAnything = useRef(false)
    const fading = useRef<number[]>([])
    const lastHeard = useRef<string | undefined>(undefined)
    const heardAnything = useRef(false)

    const flash = useCallback((ids: string[]) => {
        if (ids.length === 0) return

        setFlashing(current => new Set([...current, ...ids]))
        fading.current.push(window.setTimeout(() => setFlashing(current => {
            const rest = new Set(current)
            ids.forEach(id => rest.delete(id))
            return rest
        }), FLASH_MS))
    }, [])

    useEffect(() => () => fading.current.forEach(clearTimeout), [])

    useEffect(() => {
        const last = messages[messages.length - 1]?.id

        // The history the panel opens on is not news, however long it is.
        if (!seenAnything.current) {
            seenAnything.current = messages.length > 0
            lastSeen.current = last
            setUnread(0)
            return
        }

        if (!isOpen) {
            setUnread(unreadSince(messages, lastSeen.current))
            return
        }

        // Everything unseen lights up as it comes into view: one message while
        // the panel is open, or the whole backlog the moment it is unfolded.
        flash(idsSince(messages, lastSeen.current).filter(id => !mine.has(id)))
        lastSeen.current = last
        setUnread(0)
    }, [messages, isOpen, mine, flash])

    // Kept apart from `lastSeen`, which a folded panel holds back for the badge:
    // a message is heard as it arrives, whether or not it has been seen.
    const {playSound} = props
    useEffect(() => {
        const last = messages[messages.length - 1]?.id

        if (!heardAnything.current) {
            heardAnything.current = messages.length > 0
            lastHeard.current = last
            return
        }

        const fresh = messages.slice(messages.length - idsSince(messages, lastHeard.current).length)
        lastHeard.current = last

        // Your own message never pings. `mine` alone is not enough: the
        // broadcast of it can arrive before the answer that fills `mine` in.
        if (fresh.some(message => !mine.has(message.id) && message.authorName !== identity.name)) {
            playSound?.('chat')
        }
    }, [messages, mine, identity.name, playSound])

    if (status === 'unavailable') return null

    const onSend = (text: string) => {
        const message: OutgoingMessage = {
            authorName: identity.name,
            authorId: identity.authorId,
            countryCode: props.country.code,
            text,
        }
        return send(message)
    }

    const latest = messages[messages.length - 1]
    const waiting = !isOpen && unread > 0

    return <section className={panelClass(isOpen, waiting)} aria-label="Live chat">
        <button type="button"
                className="chat-header"
                aria-expanded={isOpen}
                aria-controls={bodyId}
                onClick={() => setIsOpen(!isOpen)}>
            <span className="chat-header-row">
                <span className="chat-header-title">Chat</span>
                {waiting &&
                    <span className="chat-badge"
                          key={unread}
                          aria-label={unread === 1 ? "1 new message" : `${unread} new messages`}>
                        {unread > UNREAD_CAP ? `${UNREAD_CAP}+` : unread}
                    </span>}
                <span className={isOpen ? "chat-chevron chat-chevron-open" : "chat-chevron"}>
                    <ChevronIcon/>
                </span>
            </span>

            {waiting && latest &&
                <span className="chat-peek"
                      key={latest.id}
                      aria-hidden="true"
                      style={authorStyle(latest.authorName, latest.authorTag)}>
                    <span className="chat-peek-author">
                        {truncate(latest.authorName, PEEK_AUTHOR_MAX_LENGTH)}
                    </span>
                    <span className="chat-peek-text">{latest.text}</span>
                </span>}
        </button>

        {isOpen && <div className="chat-body" id={bodyId}>
            <ChatLog messages={messages} loading={status === 'loading'} flashing={flashing}/>

            <ChatComposer identity={identity}
                          setName={setName}
                          failure={failure}
                          onSend={onSend}/>
        </div>}
    </section>
}

function panelClass(isOpen: boolean, waiting: boolean): string {
    return ["chat", isOpen && "chat-open", waiting && "chat-waiting"].filter(Boolean).join(" ")
}
