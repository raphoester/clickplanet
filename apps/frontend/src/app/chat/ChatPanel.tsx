import {ReactNode, useCallback, useEffect, useId, useRef, useState} from "react";
import {ChatBackend, ChatMessage, Gif, OutgoingMessage} from "../../backends/chat.ts";
import {GifChoice, GifQuery, GifSearch} from "../../backends/gifs.ts";
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
import {authorStyle} from "./authorStyle.ts";
import {nameOf, paintOf} from "../nameplates/nameplate.ts";
import {useNameplate} from "../nameplates/nameplatesStore.ts";
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
import {useChat} from "./chatStore.ts";
import {useWordFilter} from "../crazygames/wordFilter.ts";
import {t, translator} from "../../i18n/page.ts";
import {wordsOf} from "./wordsOf.ts";
import "./ChatPanel.css"

export type FlagRoom = {
    country: Country
    backend: ChatBackend
}

export type ChatPanelProps = {
    backend: ChatBackend
    flagRoom?: FlagRoom
    onLeftRoom?: () => void
    country: Country
    playSound?: PlaySound
    accountId?: string
    username?: string
    reporter: PlayerReporter
    compact?: boolean
    open?: boolean
    onOpenChange?: (open: boolean) => void
    discord?: DiscordOffer
    onSignIn?: () => void
    gifs?: GifSearch
}

type View = "world" | "flag" | "online"

const UNREAD_CAP = 99

const PEEK_AUTHOR_MAX_LENGTH = 12

const PEEK_TEXT_MAX_LENGTH = 80

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
    const [reporting, setReporting] = useState<ChatMessage>()
    const setUnread = useChat((chat) => chat.setUnread)
    const markReported = useChat((chat) => chat.markReported)
    const bodyId = useId()
    const panel = useRef<HTMLElement>(null)
    const {startResize, resetSize} = useChatSize(panel)

    const {accountId, username, playSound, flagRoom, onSignIn} = props
    const locked = onSignIn !== undefined
    const world = useChatRoom({
        backend: props.backend, accountId, playSound,
        seeing: looking && (view === "world" || (view === "online" && online === undefined)),
    })
    const flag = useChatRoom({
        backend: flagRoom?.backend, accountId, playSound: locked ? undefined : playSound, onLeft: props.onLeftRoom,
        seeing: looking && view === "flag" && !locked,
    })
    const identity = useChatIdentity()
    // A guest learns its name from the first message it sends.
    const guestName = useNameplate(world.ownAccount ?? flag.ownAccount ?? "")?.name

    const flagOpen = flagRoom !== undefined && flag.status !== 'unavailable'
    const flagRead = flagOpen && !locked
    const flagCountry = locked ? flagRoom?.country ?? props.country : flagOpen ? flagRoom.country : undefined
    const shown: View = view === "flag" && !flagCountry ? "world" : view

    useEffect(() => {
        if (view === "flag" && !flagCountry) setView("world")
    }, [view, flagCountry])

    const unread = world.unread + (flagRead ? flag.unread : 0)

    useEffect(() => {
        setUnread(unread)
    }, [unread, setUnread])

    // The tab's badge and the reports go with the panel.
    useEffect(() => () => useChat.setState(useChat.getInitialState(), true), [])

    // A copy each time: Reply pressed again on the same message puts the focus back in the box.
    const onReply = useCallback((message: ChatMessage) => setReplyingTo({...message}), [])

    const {reporter} = props

    if (world.status === 'unavailable') return null

    const room: ChatRoomState = shown === "flag" ? flag : world

    const choose = (next: View) => {
        if (next !== shown) setReplyingTo(undefined)
        setView(next)
    }

    const send = async (text: string, gif?: Gif) => {
        const target = replyingTo
        const message: OutgoingMessage = {
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

    const tabs = (online !== undefined || flagCountry) && <div className="chat-tabs" role="tablist" aria-label={t("chat.title")}>
        <RoomTab on={shown === "world"} name={flagCountry ? t("chat.world") : t("chat.title")} unread={world.unread}
                 onClick={() => choose("world")}/>
        {flagCountry && <RoomTab on={shown === "flag"} name={translator().country(flagCountry.code)} flag={flagCountry.code}
                                 unread={flag.unread} locked={locked} onClick={() => choose("flag")}/>}
        {online !== undefined && <button type="button"
                                         role="tab"
                                         aria-selected={shown === "online"}
                                         aria-label={t("chat.playersOnline", {count: online})}
                                         className={shown === "online" ? "chat-tab chat-tab--on" : "chat-tab"}
                                         onClick={() => choose("online")}>
            {t("chat.online", {count: online})}
        </button>}
    </div>

    const body: ReactNode = shown === "online" && online !== undefined
        ? <div className="chat-players"><PlayersPanel/></div>
        : shown === "flag" && flagCountry && onSignIn
        ? <LockedRoom country={flagCountry} unread={flag.unread} onSignIn={onSignIn}/>
        : <>
            <DiscordBar discord={props.discord} flagName={translator().country(props.country.code)}/>

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
                     onReport={setReporting}
                     own={room.own}/>

            <ChatComposer username={username}
                          guestName={username === undefined ? guestName : undefined}
                          failure={room.failure}
                          mutedUntil={room.mutedUntil}
                          replyingTo={replyingTo}
                          onCancelReply={() => setReplyingTo(undefined)}
                          onSend={text => send(text)}
                          gifs={gifOffer}
                          onSendGif={onSendGif}/>

            {reporting && <ReportMessageModal key={reporting.id}
                                              message={reporting}
                                              reporter={reporter}
                                              onSent={markReported}
                                              onClose={() => setReporting(undefined)}/>}
        </>

    const toast = newest(world.toast, flagRead ? flag.toast : undefined)

    if (props.compact) {
        if (isOpen) {
            return <Sheet title={t("chat.liveChat")}
                          head={tabs ?? <span className="chat-sheet-title">{t("chat.title")}</span>}
                          className="chat-sheet"
                          onClose={() => setOpen(false)}>
                {body}
            </Sheet>
        }

        return toast ? <ChatToast key={toast.id} message={toast} onOpen={() => setOpen(true)}/> : null
    }

    const latest = newest(
        world.unread > 0 ? world.messages[world.messages.length - 1] : undefined,
        flagRead && flag.unread > 0 ? flag.messages[flag.messages.length - 1] : undefined,
    )
    const waiting = !isOpen && unread > 0

    return <section ref={panel} className={panelClass(isOpen, waiting)} aria-label={t("chat.liveChat")}>
        {isOpen && RESIZE_EDGES.map(edge =>
            <div key={edge}
                 className={`chat-resize chat-resize-${edge}`}
                 aria-hidden="true"
                 title={t("chat.resize")}
                 onPointerDown={event => startResize(edge, event)}
                 onDoubleClick={resetSize}/>)}

        {isOpen
            ? <div className="chat-header chat-header-open">
                {tabs ?? <span className="chat-header-title">{t("chat.title")}</span>}
                <button type="button"
                        className="icon-button chat-fold"
                        aria-label={t("chat.fold")}
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
                    <span className="chat-header-title">{t("chat.title")}</span>
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

                {waiting && latest && <ChatPeek key={latest.id} message={latest}/>}
            </button>}

        {isOpen && <div className="chat-body" id={bodyId}>{body}</div>}
    </section>
}

function ChatToast({message, onOpen}: {message: ChatMessage, onOpen: () => void}) {
    const words = useWordFilter()
    const nameplate = useNameplate(message.authorAccountId)
    const name = words.masked(nameOf(nameplate))

    return <button type="button"
                   className="chat-toast"
                   style={authorStyle(paintOf(nameplate))}
                   aria-label={t("chat.openWith", {player: name, message: words.masked(wordsOf(message))})}
                   onClick={onOpen}>
        <span className="chat-toast-author" aria-hidden="true">{truncate(name, PEEK_AUTHOR_MAX_LENGTH)}</span>
        <span className="chat-toast-text" aria-hidden="true">{truncate(words.masked(wordsOf(message)), PEEK_TEXT_MAX_LENGTH)}</span>
    </button>
}

function ChatPeek({message}: {message: ChatMessage}) {
    const words = useWordFilter()
    const nameplate = useNameplate(message.authorAccountId)

    return <span className="chat-peek" aria-hidden="true" style={authorStyle(paintOf(nameplate))}>
        <span className="chat-peek-author">{truncate(words.masked(nameOf(nameplate)), PEEK_AUTHOR_MAX_LENGTH)}</span>
        <span className="chat-peek-text">{words.masked(wordsOf(message))}</span>
    </span>
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
                   aria-label={missed ? t("chat.roomUnread", {room: name, count: unread}) : name}
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
    return unread > UNREAD_CAP ? `${translator().number(UNREAD_CAP)}+` : translator().number(unread)
}

function panelClass(isOpen: boolean, waiting: boolean): string {
    return ["chat", "panel", isOpen && "chat-open", waiting && "chat-waiting"].filter(Boolean).join(" ")
}
