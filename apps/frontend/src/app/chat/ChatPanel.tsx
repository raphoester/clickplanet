import {ReactNode, useCallback, useEffect, useId, useRef, useState} from "react";
import {ChatBackend, ChatMessage, OutgoingMessage} from "../../backends/chat.ts";
import {PlayerLine, RosterEntry} from "../../backends/player.ts";
import {Country} from "../../domain/countries.ts";
import {idsSince, unseenAfter} from "../../domain/chatLog.ts";
import {ChevronIcon} from "../components/icons.tsx";
import {opensFolded} from "../compact.ts";
import {truncate} from "../truncate.ts";
import {usePageVisible} from "../usePageVisible.ts";
import {PlaySound} from "../sound/soundPlayer.ts";
import Sheet from "../hud/Sheet.tsx";
import PlayersPanel from "../players/PlayersPanel.tsx";
import {authorOf, authorStyle} from "./authorStyle.ts";
import {RESIZE_EDGES} from "./chatSize.ts";
import ChatComposer from "./ChatComposer.tsx";
import ChatLog from "./ChatLog.tsx";
import {useChat} from "./useChat.ts";
import {useChatIdentity} from "./useChatIdentity.ts";
import {useChatSize} from "./useChatSize.ts";
import {useSeenMark} from "./useSeenMark.ts";
import "./ChatPanel.css"

export type ChatPanelProps = {
    backend?: ChatBackend
    country: Country
    playSound?: PlaySound
    username?: string
    onOpenPlayer?: (player: PlayerLine) => void
    players?: readonly RosterEntry[]
    compact?: boolean
    open?: boolean
    onOpenChange?: (open: boolean) => void
    onUnread?: (unread: number) => void
}

type View = "chat" | "online"

const UNREAD_CAP = 99

const PEEK_AUTHOR_MAX_LENGTH = 12

const PEEK_TEXT_MAX_LENGTH = 80

const FLASH_MS = 1600

const TOAST_MS = 4000

const NOTHING: ReadonlySet<string> = new Set()

export default function ChatPanel(props: ChatPanelProps) {
    const [ownOpen, setOwnOpen] = useState(() => !opensFolded())
    const isOpen = props.open ?? ownOpen
    const {onOpenChange} = props
    const setOpen = useCallback((open: boolean) => onOpenChange ? onOpenChange(open) : setOwnOpen(open), [onOpenChange])
    const [view, setView] = useState<View>("chat")
    const players = props.players
    const visible = usePageVisible()
    const seeing = isOpen && (view === "chat" || !players) && visible

    const [toast, setToast] = useState<ChatMessage>()
    const [flashing, setFlashing] = useState<ReadonlySet<string>>(NOTHING)
    const bodyId = useId()
    const panel = useRef<HTMLElement>(null)
    const {startResize, resetSize} = useChatSize(panel)

    const {username} = props
    const {messages, announcements, mine, displayName, seenAtLoad, status, failure, mutedUntil, send, react} =
        useChat({backend: props.backend, username})
    const identity = useChatIdentity()
    const markSeen = useSeenMark(props.backend, seenAtLoad?.kept ? seenAtLoad.until : 0)
    const [seenLater, setSeenLater] = useState(0)
    const seenUntil = seenAtLoad === undefined ? undefined : Math.max(seenAtLoad.until, seenLater)

    const lastShown = useRef<string | undefined>(undefined)
    const shownAnything = useRef(false)
    const newestId = useRef<string | undefined>(undefined)
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
        // So the next visit has a mark to count from, even if the chat is never opened in this one.
        if (seenAtLoad && !seenAtLoad.kept) markSeen(seenAtLoad.until)
    }, [seenAtLoad, markSeen])

    const onSeen = useCallback((at: number) => {
        setSeenLater(current => Math.max(current, at))
        markSeen(at)
    }, [markSeen])

    const unread = seenUntil === undefined ? 0 : unseenAfter(messages, announcements, seenUntil,
        message => mine.has(message.id) || message.authorName === displayName)

    useEffect(() => {
        const last = messages[messages.length - 1]

        if (!shownAnything.current) {
            shownAnything.current = messages.length > 0
            lastShown.current = last?.id
            return
        }
        if (!seeing) return

        flash(idsSince(messages, lastShown.current).filter(id => !mine.has(id)))
        lastShown.current = last?.id
    }, [messages, seeing, mine, flash])

    useEffect(() => {
        if (seeing) setToast(undefined)
    }, [seeing])

    useEffect(() => {
        const last = messages[messages.length - 1]
        if (seenUntil === undefined || !last || last.id === newestId.current) return
        newestId.current = last.id

        if (seeing || last.sentAt <= seenUntil || mine.has(last.id) || last.authorName === displayName) return
        setToast(last)
    }, [seenUntil, messages, seeing, mine, displayName])

    useEffect(() => {
        if (!toast) return
        const timer = setTimeout(() => setToast(undefined), TOAST_MS)
        return () => clearTimeout(timer)
    }, [toast])

    const {onUnread} = props
    useEffect(() => {
        onUnread?.(unread)
    }, [unread, onUnread])

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

        // The name check too: an own message's broadcast can land before `mine` holds its id.
        if (fresh.some(message => !mine.has(message.id) && message.authorName !== displayName)) {
            playSound?.('chat')
        }
    }, [messages, mine, displayName, playSound])

    if (status === 'unavailable') return null

    const onSend = (text: string) => {
        const message: OutgoingMessage = {
            authorId: identity.authorId,
            countryCode: props.country.code,
            text,
        }
        return send(message)
    }

    const tabs = players && <div className="chat-tabs" role="tablist" aria-label="Chat">
        <button type="button"
                role="tab"
                aria-selected={view === "chat"}
                className={view === "chat" ? "chat-tab chat-tab--on" : "chat-tab"}
                onClick={() => setView("chat")}>
            Chat
        </button>
        <button type="button"
                role="tab"
                aria-selected={view === "online"}
                aria-label={players.length === 1 ? "1 player online" : `${players.length} players online`}
                className={view === "online" ? "chat-tab chat-tab--on" : "chat-tab"}
                onClick={() => setView("online")}>
            Online · {players.length}
        </button>
    </div>

    const body: ReactNode = view === "online" && players
        ? <div className="chat-players"><PlayersPanel entries={players} onOpenPlayer={props.onOpenPlayer}/></div>
        : <>
            <ChatLog messages={messages}
                     announcements={announcements}
                     loading={status === 'loading'}
                     seenUntil={seenUntil}
                     onSeen={onSeen}
                     watching={visible}
                     flashing={flashing}
                     onOpenPlayer={props.onOpenPlayer}
                     onReact={(messageId, reaction, on) =>
                         react({messageId, reaction, on})}/>

            <ChatComposer username={username}
                          guestName={username === undefined ? displayName : undefined}
                          failure={failure}
                          mutedUntil={mutedUntil}
                          onSend={onSend}/>
        </>

    if (props.compact) {
        if (isOpen) {
            return <Sheet title="Live chat"
                          head={tabs ?? <span className="chat-sheet-title">Chat</span>}
                          className="chat-sheet"
                          onClose={() => setOpen(false)}>
                {body}
            </Sheet>
        }

        return toast ? <button type="button"
                               key={toast.id}
                               className="chat-toast"
                               style={authorStyle(authorOf(toast))}
                               aria-label={`Open the chat: ${toast.authorName}, ${toast.text}`}
                               onClick={() => setOpen(true)}>
            <span className="chat-toast-author" aria-hidden="true">
                {truncate(toast.authorName, PEEK_AUTHOR_MAX_LENGTH)}
            </span>
            <span className="chat-toast-text" aria-hidden="true">{truncate(toast.text, PEEK_TEXT_MAX_LENGTH)}</span>
        </button> : null
    }

    const latest = messages[messages.length - 1]
    const waiting = !isOpen && unread > 0

    return <section ref={panel} className={panelClass(isOpen, waiting)} aria-label="Live chat">
        {isOpen && RESIZE_EDGES.map(edge =>
            <div key={edge}
                 className={`chat-resize chat-resize-${edge}`}
                 aria-hidden="true"
                 title="Drag to resize, double-click to reset"
                 onPointerDown={event => startResize(edge, event)}
                 onDoubleClick={resetSize}/>)}

        {isOpen
            ? <div className="chat-header chat-header-open">
                {tabs ?? <span className="chat-header-title">Chat</span>}
                <button type="button"
                        className="icon-button chat-fold"
                        aria-label="Fold the chat"
                        aria-expanded={true}
                        aria-controls={bodyId}
                        onClick={() => setOpen(false)}>
                    <ChevronIcon/>
                </button>
            </div>
            : <button type="button"
                      className="chat-header"
                      aria-expanded={false}
                      onClick={() => setOpen(true)}>
                <span className="chat-header-row">
                    <span className="chat-header-title">Chat</span>
                    {waiting &&
                        <span className="chip chat-badge"
                              key={unread}
                              aria-label={unread === 1 ? "1 new message" : `${unread} new messages`}>
                            {unread > UNREAD_CAP ? `${UNREAD_CAP}+` : unread}
                        </span>}
                    <span className="chat-chevron">
                        <ChevronIcon/>
                    </span>
                </span>

                {waiting && latest &&
                    <span className="chat-peek"
                          key={latest.id}
                          aria-hidden="true"
                          style={authorStyle(authorOf(latest))}>
                        <span className="chat-peek-author">
                            {truncate(latest.authorName, PEEK_AUTHOR_MAX_LENGTH)}
                        </span>
                        <span className="chat-peek-text">{latest.text}</span>
                    </span>}
            </button>}

        {isOpen && <div className="chat-body" id={bodyId}>{body}</div>}
    </section>
}

function panelClass(isOpen: boolean, waiting: boolean): string {
    return ["chat", "panel", isOpen && "chat-open", waiting && "chat-waiting"].filter(Boolean).join(" ")
}
