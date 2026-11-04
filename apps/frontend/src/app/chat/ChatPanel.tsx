import {ReactNode, useCallback, useEffect, useId, useRef, useState} from "react";
import {ChatBackend, ChatMessage, Gif, OutgoingMessage} from "../../backends/chat.ts";
import {GifChoice, GifQuery, GifSearch} from "../../backends/gifs.ts";
import {wordsOf} from "../../domain/chatLog.ts";
import {PlayerReporter} from "../../backends/support.ts";
import {Country} from "../../domain/countries.ts";
import {ChevronIcon} from "../components/icons.tsx";
import CountryFlag from "../components/CountryFlag.tsx";
import {opensFolded} from "../compact.ts";
import {truncate} from "../truncate.ts";
import {usePageVisible} from "../usePageVisible.ts";
import {PlaySound} from "../sound/soundPlayer.ts";
import Sheet from "../hud/Sheet.tsx";
import PlayersPanel from "../players/PlayersPanel.tsx";
import {onlineCount, useRoster} from "../players/rosterStore.ts";
import {authorOf, authorStyle} from "./authorStyle.ts";
import {RESIZE_EDGES} from "./chatSize.ts";
import ChatComposer from "./ChatComposer.tsx";
import ChatLog from "./ChatLog.tsx";
import ReportMessageModal from "../support/ReportMessageModal.tsx";
import DiscordBar from "./DiscordBar.tsx";
import LockedRoom from "./LockedRoom.tsx";
import {newMessages} from "./newMessages.ts";
import {DiscordOffer} from "../alerts/useDiscordAlerts.ts";
import {ChatRoomState, useChatRoom} from "./useChatRoom.ts";
import {useChatIdentity} from "./useChatIdentity.ts";
import {useChatSize} from "./useChatSize.ts";
import {useWordFilter} from "../crazygames/wordFilter.ts";
import "./ChatPanel.css"

export type FlagRoom = {
    country: Country
    backend: ChatBackend
}

export type ChatPanelProps = {
    backend?: ChatBackend
    flagRoom?: FlagRoom
    onLeftRoom?: () => void
    country: Country
    playSound?: PlaySound
    accountId?: string
    username?: string
    reporter?: PlayerReporter
    compact?: boolean
    open?: boolean
    onOpenChange?: (open: boolean) => void
    onUnread?: (unread: number) => void
    discord?: DiscordOffer
    onSignIn?: () => void
    gifs?: GifSearch
}

type View = "world" | "flag" | "online"

const UNREAD_CAP = 99

const PEEK_AUTHOR_MAX_LENGTH = 12

const PEEK_TEXT_MAX_LENGTH = 80

const NOTHING: ReadonlySet<string> = new Set()

export default function ChatPanel(props: ChatPanelProps) {
    const [ownOpen, setOwnOpen] = useState(() => !opensFolded())
    const isOpen = props.open ?? ownOpen
    const {onOpenChange} = props
    const setOpen = useCallback((open: boolean) => onOpenChange ? onOpenChange(open) : setOwnOpen(open), [onOpenChange])
    const [view, setView] = useState<View>("world")
    const online = useRoster(onlineCount)
    const visible = usePageVisible()
    const looking = isOpen && visible

    const [replyingTo, setReplyingTo] = useState<ChatMessage>()
    const [reported, setReported] = useState<ReadonlySet<string>>(NOTHING)
    const [reporting, setReporting] = useState<ChatMessage>()
    const bodyId = useId()
    const panel = useRef<HTMLElement>(null)
    const {startResize, resetSize} = useChatSize(panel)

    const {accountId, username, playSound, flagRoom, onSignIn} = props
    const locked = onSignIn !== undefined
    const world = useChatRoom({
        backend: props.backend, accountId, username, playSound,
        seeing: looking && (view === "world" || (view === "online" && online === undefined)),
    })
    const flag = useChatRoom({
        backend: flagRoom?.backend, accountId, username, playSound: locked ? undefined : playSound, onLeft: props.onLeftRoom,
        seeing: looking && view === "flag" && !locked,
    })
    const words = useWordFilter()
    const identity = useChatIdentity()

    const flagOpen = flagRoom !== undefined && flag.status !== 'unavailable'
    const flagRead = flagOpen && !locked
    const flagCountry = locked ? flagRoom?.country ?? props.country : flagOpen ? flagRoom.country : undefined
    const shown: View = view === "flag" && !flagCountry ? "world" : view

    useEffect(() => {
        if (view === "flag" && !flagCountry) setView("world")
    }, [view, flagCountry])

    const unread = world.unread + (flagRead ? flag.unread : 0)

    const {onUnread} = props
    useEffect(() => {
        onUnread?.(unread)
    }, [unread, onUnread])

    // A copy each time: Reply pressed again on the same message puts the focus back in the box.
    const onReply = useCallback((message: ChatMessage) => setReplyingTo({...message}), [])

    const {reporter} = props
    const onReported = useCallback((message: ChatMessage) =>
        setReported(current => new Set([...current, message.id])), [])

    if (world.status === 'unavailable') return null

    const room: ChatRoomState = shown === "flag" ? flag : world

    const choose = (next: View) => {
        if (next !== shown) setReplyingTo(undefined)
        setView(next)
    }

    const send = async (text: string, gif?: Gif) => {
        const target = replyingTo
        const message: OutgoingMessage = {
            authorId: identity.authorId,
            countryCode: props.country.code,
            text,
            replyTo: target?.id,
            gif,
        }

        setReplyingTo(undefined)
        const failed = await room.send(message)
        if (failed !== undefined && failed !== 'gone') setReplyingTo(current => current ?? target)
        return failed === undefined
    }

    const {gifs} = props
    const gifOffer = gifs && {search: gifs, locale: props.country.code, customer: identity.authorId, onSignIn}
    const onSendGif = async (choice: GifChoice, query: GifQuery) => {
        const sent = await send("", choice.gif)
        if (sent) gifs?.markShared(choice, query)
        return sent
    }

    const tabs = (online !== undefined || flagCountry) && <div className="chat-tabs" role="tablist" aria-label="Chat">
        <RoomTab on={shown === "world"} name={flagCountry ? "World" : "Chat"} unread={world.unread}
                 onClick={() => choose("world")}/>
        {flagCountry && <RoomTab on={shown === "flag"} name={flagCountry.name} flag={flagCountry.code}
                                 unread={flag.unread} locked={locked} onClick={() => choose("flag")}/>}
        {online !== undefined && <button type="button"
                                         role="tab"
                                         aria-selected={shown === "online"}
                                         aria-label={online === 1 ? "1 player online" : `${online} players online`}
                                         className={shown === "online" ? "chat-tab chat-tab--on" : "chat-tab"}
                                         onClick={() => choose("online")}>
            Online · {online}
        </button>}
    </div>

    const body: ReactNode = shown === "online" && online !== undefined
        ? <div className="chat-players"><PlayersPanel/></div>
        : shown === "flag" && flagCountry && onSignIn
        ? <LockedRoom country={flagCountry} unread={flag.unread} onSignIn={onSignIn}/>
        : <>
            <DiscordBar discord={props.discord} flagName={props.country.name}/>

            <ChatLog key={shown === "flag" ? `flag-${flagRoom?.country.code}` : "world"}
                     messages={room.messages}
                     announcements={room.announcements}
                     loading={room.status === 'loading'}
                     seenUntil={room.seenUntil}
                     onSeen={room.onSeen}
                     watching={visible}
                     flashing={room.flashing}
                     onReact={(messageId, reaction, on) =>
                         room.react({messageId, reaction, on})}
                     onReply={onReply}
                     onReport={reporter && setReporting}
                     reported={reported}
                     own={room.own}/>

            <ChatComposer username={username}
                          guestName={username === undefined ? world.displayName ?? flag.displayName : undefined}
                          failure={room.failure}
                          mutedUntil={room.mutedUntil}
                          replyingTo={replyingTo}
                          onCancelReply={() => setReplyingTo(undefined)}
                          onSend={text => send(text)}
                          gifs={gifOffer}
                          onSendGif={onSendGif}/>

            {reporting && reporter && <ReportMessageModal key={reporting.id}
                                                          message={reporting}
                                                          reporter={reporter}
                                                          onSent={onReported}
                                                          onClose={() => setReporting(undefined)}/>}
        </>

    const toast = newest(world.toast, flagRead ? flag.toast : undefined)

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
                               aria-label={`Open the chat: ${words.masked(toast.authorName)}, ${words.masked(wordsOf(toast))}`}
                               onClick={() => setOpen(true)}>
            <span className="chat-toast-author" aria-hidden="true">
                {truncate(words.masked(toast.authorName), PEEK_AUTHOR_MAX_LENGTH)}
            </span>
            <span className="chat-toast-text" aria-hidden="true">{truncate(words.masked(wordsOf(toast)), PEEK_TEXT_MAX_LENGTH)}</span>
        </button> : null
    }

    const latest = newest(
        world.unread > 0 ? world.messages[world.messages.length - 1] : undefined,
        flagRead && flag.unread > 0 ? flag.messages[flag.messages.length - 1] : undefined,
    )
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
                              aria-label={newMessages(unread)}>
                            {badgeCount(unread)}
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
                            {truncate(words.masked(latest.authorName), PEEK_AUTHOR_MAX_LENGTH)}
                        </span>
                        <span className="chat-peek-text">{words.masked(wordsOf(latest))}</span>
                    </span>}
            </button>}

        {isOpen && <div className="chat-body" id={bodyId}>{body}</div>}
    </section>
}

type RoomTabProps = {
    on: boolean
    name: string
    flag?: string
    unread: number
    locked?: boolean
    onClick: () => void
}

function RoomTab({on, name, flag, unread, locked, onClick}: RoomTabProps) {
    const missed = (!on || locked === true) && unread > 0

    return <button type="button"
                   role="tab"
                   aria-selected={on}
                   aria-label={missed ? `${name}, ${newMessages(unread)}` : name}
                   title={flag && name}
                   className={["chat-tab", on && "chat-tab--on", flag && "chat-tab--flag"].filter(Boolean).join(" ")}
                   onClick={onClick}>
        {flag ? <><CountryFlag code={flag}/><span className="chat-tab-name">{name}</span></> : name}
        {missed && <span className="chip chat-badge" key={unread} aria-hidden="true">{badgeCount(unread)}</span>}
    </button>
}

function newest(...messages: (ChatMessage | undefined)[]): ChatMessage | undefined {
    return messages.reduce<ChatMessage | undefined>(
        (latest, message) => message && (!latest || message.sentAt > latest.sentAt) ? message : latest, undefined)
}

function badgeCount(unread: number): string {
    return unread > UNREAD_CAP ? `${UNREAD_CAP}+` : `${unread}`
}

function panelClass(isOpen: boolean, waiting: boolean): string {
    return ["chat", "panel", isOpen && "chat-open", waiting && "chat-waiting"].filter(Boolean).join(" ")
}
