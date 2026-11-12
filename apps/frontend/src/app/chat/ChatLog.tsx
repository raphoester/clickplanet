import {Fragment, ReactNode, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState} from "react";
import {BombAnnouncement, ChampionAnnouncement, ChatAnnouncement, ChatMessage, ChatQuote, FlagMissionAnnouncement,
    FortifyAnnouncement, KnockOutAnnouncement, MuteAnnouncement, MuteKind, Reaction, RoundAnnouncement} from "../../backends/chat.ts";
import AdminCrown from "../components/AdminCrown.tsx";
import CountryFlag from "../components/CountryFlag.tsx";
import Streak from "../badges/Streak.tsx";
import WornTitle from "../badges/WornTitle.tsx";
import {ChevronIcon, ReplyIcon, ReportIcon} from "../components/icons.tsx";
import {ChatLogEntry, interleave, newestAt, startsGroup} from "../../domain/chatLog.ts";
import {fitted} from "../../domain/gifs.ts";
import {messageParts} from "../../domain/chatLinks.ts";
import {muteLengthOf, MuteUnit} from "../../domain/mute.ts";
import {medalOf} from "../../domain/roundReveal.ts";
import {truncate} from "../truncate.ts";
import {authorStyle, quoteStyle} from "./authorStyle.ts";
import {lineOf, nameOf, paintOf} from "../nameplates/nameplate.ts";
import {useNameplate} from "../nameplates/nameplatesStore.ts";
import {Nameplate} from "../../backends/nameplates.ts";
import QuoteLine from "./QuoteLine.tsx";
import ReactionBar, {AddReactionButton} from "./ReactionBar.tsx";
import {useWordFilter} from "../crazygames/wordFilter.ts";
import {usePlayerCard} from "../players/playerCardStore.ts";
import {useChat} from "./chatStore.ts";
import {partsOf} from "../missions/orderParts.tsx";
import {t, translator} from "../../i18n/page.ts";
import {rich} from "../../i18n/rich.tsx";
import {Key} from "../../i18n/translator.ts";

export type ChatLogProps = {
    messages: ChatMessage[]
    announcements?: ChatAnnouncement[]
    loading: boolean
    flashing?: ReadonlySet<string>
    onReact?: (messageId: string, reaction: Reaction, on: boolean) => void
    onReply?: (message: ChatMessage) => void
    onReport?: (message: ChatMessage) => void
    own?: (message: ChatMessage) => boolean
    seenUntil?: number
    onSeen?: (at: number) => void
    watching?: boolean
}

const AUTHOR_MAX_LENGTH = 16

const GIF_BOX = {width: 200, height: 200}

const CLOCK: Intl.DateTimeFormatOptions = {hour: "2-digit", minute: "2-digit"}

const MUTE_LINES: Record<MuteKind, Record<MuteUnit, Key>> = {
    chat: {
        week: "mute.chat.week",
        day: "mute.chat.day",
        hour: "mute.chat.hour",
        minute: "mute.chat.minute",
        second: "mute.chat.second",
    },
    gifs: {
        week: "mute.gifs.week",
        day: "mute.gifs.day",
        hour: "mute.gifs.hour",
        minute: "mute.gifs.minute",
        second: "mute.gifs.second",
    },
}

const PINNED_SLACK_PX = 40

const SEEN_SLACK_PX = 4

const CONTEXT_ABOVE_UNSEEN_PX = 24

const SPOTLIGHT_MS = 1600

const NO_ANNOUNCEMENTS: ChatAnnouncement[] = []

export default function ChatLog(props: ChatLogProps) {
    const scroll = useRef<HTMLDivElement>(null)
    const pinned = useRef(true)
    const placed = useRef(false)
    const reported = useRef(0)
    const [behind, setBehind] = useState(false)
    const [picking, setPicking] = useState<string | undefined>(undefined)
    const [spotlit, setSpotlit] = useState<string | undefined>(undefined)
    const dimming = useRef<number>()
    const reportedIds = useChat((chat) => chat.reported)
    const shown = useMemo(() => new Set(props.messages.map(message => message.id)), [props.messages])
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

    useEffect(() => () => window.clearTimeout(dimming.current), [])

    const jumpTo = (id: string) => {
        const lines = scroll.current?.querySelectorAll<HTMLElement>("[data-message]") ?? []
        const line = [...lines].find(line => line.dataset.message === id)
        if (!line) return

        line.scrollIntoView?.({block: "center"})
        setSpotlit(id)
        window.clearTimeout(dimming.current)
        dimming.current = window.setTimeout(() => setSpotlit(undefined), SPOTLIGHT_MS)
    }

    const jumpToLatest = () => {
        const element = scroll.current
        if (!element) return
        element.scrollTop = element.scrollHeight
        pinned.current = true
        look()
    }

    if (props.loading) {
        return <div className="chat-log chat-log-empty">
            <p>{t("chat.loading")}</p>
        </div>
    }

    if (props.messages.length === 0 && announcements.length === 0) {
        return <div className="chat-log chat-log-empty">
            <p>{t("chat.empty")}</p>
        </div>
    }

    return <div className="chat-log-shell">
        <div className="chat-log" ref={scroll} onScroll={onScroll}>
            <ul className="chat-messages" aria-live="polite">
                {interleave(props.messages, announcements).map((entry, index, entries) => {
                    const unseen = unseenAfter != null && atOf(entry) > unseenAfter
                        && (index === 0 || atOf(entries[index - 1]) <= unseenAfter)
                    return <Fragment key={entry.kind === "message" ? entry.message.id : entry.announcement.id}>
                        {unseen && <li className="chat-unseen" role="separator">{t("chat.unread")}</li>}
                        {entry.kind === "announcement"
                            ? <AnnouncementLine announcement={entry.announcement}/>
                            : <MessageLine message={entry.message}
                                           previous={entries[index - 1]}
                                           flashing={props.flashing}
                                           spotlit={spotlit === entry.message.id}
                                           picking={picking}
                                           pick={pick}
                                           jumpTo={shown.has(entry.message.replyTo?.messageId ?? "") ? jumpTo : undefined}
                                           onReact={entry.message.removed ? undefined : props.onReact}
                                           onReply={entry.message.removed ? undefined : props.onReply}
                                           onReport={props.own?.(entry.message) ? undefined : props.onReport}
                                           reported={reportedIds.has(entry.message.id)}/>}
                    </Fragment>
                })}
            </ul>
        </div>

        {behind && <button type="button" className="chat-jump" onClick={jumpToLatest}>
            {t("chat.seeNewMessages")}
            <ChevronIcon size={14}/>
        </button>}
    </div>
}

type MessageLineProps = {
    message: ChatMessage
    previous: ChatLogEntry | undefined
    flashing?: ReadonlySet<string>
    spotlit: boolean
    picking: string | undefined
    pick: (id: string) => (open: boolean) => void
    jumpTo?: (id: string) => void
    onReact?: (messageId: string, reaction: Reaction, on: boolean) => void
    onReply?: (message: ChatMessage) => void
    onReport?: (message: ChatMessage) => void
    reported: boolean
}

function MessageLine({message, previous, flashing, spotlit, picking, pick, jumpTo, ...props}: MessageLineProps) {
    const nameplate = useNameplate(message.authorAccountId)
    // Nobody to open a card of, or to report, once the account is gone.
    const author = lineOf(message.authorAccountId, message.countryCode, nameplate)
    const opens = startsGroup(previous?.kind === "message" ? previous.message : undefined, message)
    const words = useWordFilter()
    const name = words.masked(nameOf(nameplate))
    const openPlayer = usePlayerCard((card) => card.open)
    const onReport = author && !message.removed ? props.onReport : undefined

    return <li className={messageClass(flashing, message.id, opens, spotlit)}
               style={authorStyle(paintOf(nameplate))}
               data-message={message.id}
               data-at={message.sentAt}>
        {opens && <div className="chat-message-head">
            <span className="chat-message-country"
                  role="img"
                  aria-label={countryName(message.countryCode)}
                  title={countryName(message.countryCode)}>
                <CountryFlag code={message.countryCode}/>
            </span>
            {author
                ? <button type="button"
                          className="chat-message-author player-name-button"
                          title={name}
                          onClick={() => openPlayer(author)}>
                    {truncate(name, AUTHOR_MAX_LENGTH)}
                </button>
                : <span className="chat-message-author">
                    {truncate(name, AUTHOR_MAX_LENGTH)}
                </span>}
            {author?.admin && <AdminCrown size={13}/>}
            <WornTitle accountId={message.authorAccountId}/>
            <Streak accountId={message.authorAccountId}/>
            <time className="chat-message-time"
                  dateTime={new Date(message.sentAt).toISOString()}>
                {clockOf(message.sentAt)}
            </time>
        </div>}

        {message.replyTo && <Quote quote={message.replyTo} jumpTo={jumpTo}/>}

        <div className="chat-message-line">
            {message.removed
                ? <p className="chat-message-text chat-removed">{t("chat.removed")}</p>
                : message.gif
                ? <img className="chat-message-gif"
                       src={message.gif.url}
                       {...fitted(message.gif, GIF_BOX)}
                       alt={t("chat.gif")}
                       loading="lazy"
                       decoding="async"/>
                : <p className="chat-message-text">
                    {messageParts(message, guestOf(nameplate)).map((part, index) => part.kind === "link"
                        ? <a key={index} href={part.href} target="_blank" rel="noopener noreferrer nofollow ugc">{words.masked(part.text)}</a>
                        : <Fragment key={index}>{words.masked(part.text)}</Fragment>)}
                </p>}
            {props.onReact && <AddReactionButton messageId={message.id}
                                                 picking={picking === message.id}
                                                 setPicking={pick(message.id)}/>}
            {props.onReply && <button type="button"
                                      className="chat-reply-add"
                                      aria-label={t("chat.replyTo", {player: name})}
                                      title={t("chat.reply")}
                                      onClick={() => props.onReply?.(message)}>
                <ReplyIcon/>
            </button>}
            {onReport && (props.reported
                ? <span className="chat-report-add chat-report-done" role="img" aria-label={t("chat.reported")} title={t("chat.reported")}>
                    <ReportIcon/>
                </span>
                : <button type="button"
                          className="chat-report-add"
                          aria-label={t("chat.reportMessageFrom", {player: name})}
                          title={t("chat.report")}
                          onClick={() => onReport(message)}>
                    <ReportIcon/>
                </button>)}
        </div>

        <ReactionBar messageId={message.id}
                     reactions={message.reactions}
                     onReact={props.onReact && ((reaction, on) => props.onReact?.(message.id, reaction, on))}
                     picking={picking === message.id}
                     setPicking={pick(message.id)}/>
    </li>
}

function Quote({quote, jumpTo}: {quote: ChatQuote, jumpTo?: (id: string) => void}) {
    const style = quoteStyle(paintOf(useNameplate(quote.authorAccountId)))
    const line = <QuoteLine quote={quote} lead="inReplyTo"/>

    return jumpTo
        ? <button type="button"
                  className="chat-quote"
                  style={style}
                  onClick={() => jumpTo(quote.messageId)}>
            {line}
        </button>
        : <p className="chat-quote" style={style}>{line}</p>
}

function AnnouncementLine({announcement}: {announcement: ChatAnnouncement}) {
    switch (announcement.kind) {
        case "bomb":
            return <BombLine announcement={announcement}/>
        case "fortify":
            return <FortifyLine announcement={announcement}/>
        case "mute":
            return <MuteLine announcement={announcement}/>
        case "round":
            return <RoundLine announcement={announcement}/>
        case "champion":
            return <ChampionLine announcement={announcement}/>
        case "flag_mission":
            return <FlagMissionLine announcement={announcement}/>
        case "knock_out":
            return <KnockOutLine announcement={announcement}/>
    }
}

function KnockOutLine({announcement}: {announcement: KnockOutAnnouncement}) {
    return <li className="chat-announcement" data-at={announcement.announcedAt}>
        <span className="chat-announcement-icon" aria-hidden="true">👑</span>
        <CountryFlag code={announcement.by}/>
        <span className="chat-announcement-text">
            {rich("announcement.knockedOut", {
                winner: <strong>{countryName(announcement.by)}</strong>,
                loser: <strong>{countryName(announcement.country)}</strong>,
            })}
        </span>
        <AnnouncedAt at={announcement.announcedAt}/>
    </li>
}

// The words of the moment the flag's own players see.
function FlagMissionLine({announcement}: {announcement: FlagMissionAnnouncement}) {
    return <li className="chat-announcement" data-at={announcement.announcedAt}>
        <span className="chat-announcement-icon" aria-hidden="true">🎯</span>
        <CountryFlag code={announcement.country}/>
        <span className="chat-announcement-text">
            {announcement.order
                ? rich("announcement.flagMissionOrder", {
                    country: <strong>{countryName(announcement.country)}</strong>,
                    order: partsOf(announcement.order).line,
                    points: announcement.points,
                })
                : rich("announcement.flagMission", {
                    country: <strong>{countryName(announcement.country)}</strong>,
                    points: announcement.points,
                })}
        </span>
        <AnnouncedAt at={announcement.announcedAt}/>
    </li>
}

function ChampionLine({announcement}: {announcement: ChampionAnnouncement}) {
    return <li className="chat-announcement chat-announcement-champion" data-at={announcement.announcedAt}>
        <span className="chat-announcement-icon" aria-hidden="true">🏆</span>
        {announcement.countries.map((code) => <CountryFlag key={code} code={code}/>)}
        <strong className="chat-announcement-text">
            {t("announcement.champion", {
                countries: translator().list(announcement.countries.map(countryName)),
                count: announcement.countries.length,
                season: announcement.season,
            })}
        </strong>
        <AnnouncedAt at={announcement.announcedAt}/>
    </li>
}

function FortifyLine({announcement}: {announcement: FortifyAnnouncement}) {
    return <li className="chat-announcement" data-at={announcement.announcedAt}>
        <span className="chat-announcement-icon" aria-hidden="true">🛡️</span>
        <CountryFlag code={announcement.country}/>
        <span className="chat-announcement-text">
            {rich("announcement.fortified", {
                country: <strong>{countryName(announcement.country)}</strong>,
                landmass: translator().landmass(announcement.landmass, announcement.ground),
            })}
        </span>
        <AnnouncedAt at={announcement.announcedAt}/>
    </li>
}

function RoundLine({announcement}: {announcement: RoundAnnouncement}) {
    return <li className="chat-announcement chat-announcement-round" data-at={announcement.announcedAt}>
        <span className="chat-round-head">
            <span className="chat-announcement-icon" aria-hidden="true">{announcement.own ? "⚔️" : "🏁"}</span>
            <strong>{roundOverOf(announcement)}</strong>
            {announcement.own && announcement.times > 1 && <span className="chat-round-times">×{translator().number(announcement.times)}</span>}
            <AnnouncedAt at={announcement.announcedAt}/>
        </span>
        {announcement.podium.length > 0 && <ol className="chat-round-podium">
            {announcement.podium.map((place) => <li key={place.country} className="chat-round-place">
                <span aria-hidden="true">{medalOf(place.rank)}</span>
                <CountryFlag code={place.country}/>
                <span className="chat-round-country">{countryName(place.country)}</span>
                <span className="chat-round-points">{translator().number(place.points, {signDisplay: "always"})}</span>
            </li>)}
        </ol>}
    </li>
}

function MuteLine({announcement}: {announcement: MuteAnnouncement}) {
    const words = useWordFilter()
    const length = muteLengthOf(announcement.seconds)

    return <li className="chat-announcement" data-at={announcement.announcedAt}>
        <span className="chat-announcement-icon" aria-hidden="true">🔇</span>
        <span className="chat-announcement-text">
            {rich(MUTE_LINES[announcement.muteKind][length.unit], {
                player: <strong>{words.masked(announcement.name)}</strong>,
                count: length.count,
            })}
        </span>
        <AnnouncedAt at={announcement.announcedAt}/>
    </li>
}

function BombLine({announcement}: {announcement: BombAnnouncement}) {
    return <li className="chat-announcement" data-at={announcement.announcedAt}>
        <span className="chat-announcement-icon" aria-hidden="true">
            {announcement.tile === undefined ? "🌊" : "💥"}
        </span>
        <CountryFlag code={announcement.country}/>
        <span className="chat-announcement-text">
            {bombLineOf(announcement, <strong>{countryName(announcement.country)}</strong>)}
        </span>
        <AnnouncedAt at={announcement.announcedAt}/>
    </li>
}

function bombLineOf(announcement: BombAnnouncement, country: ReactNode): ReactNode {
    if (announcement.tile === undefined) return rich("announcement.bombedOcean", {country})
    const landmass = announcement.ground === undefined ? "" : translator().landmass(announcement.landmass ?? 0, announcement.ground)
    if (landmass) return rich("announcement.bombedLand", {country, landmass})
    if (announcement.cleared === 0) return rich("announcement.bombedEmptyLand", {country})
    return rich("announcement.bombedTiles", {country, count: announcement.cleared})
}

function roundOverOf(round: RoundAnnouncement): string {
    if (!round.own) return t("announcement.dayOver", {day: round.number})
    return round.finale ? t("announcement.finalBattleOver") : t("announcement.battleOver")
}

function AnnouncedAt({at}: {at: number}) {
    return <time className="chat-announcement-time" dateTime={new Date(at).toISOString()}>
        {clockOf(at)}
    </time>
}

function clockOf(at: number): string {
    return translator().dateTime(at, CLOCK)
}

function messageClass(
    flashing: ReadonlySet<string> | undefined,
    id: string,
    opens: boolean,
    spotlit: boolean,
): string {
    return [
        "chat-message",
        opens ? "chat-message-opens" : "chat-message-cont",
        flashing?.has(id) && "chat-message-new",
        spotlit && "chat-message-spotlit",
    ].filter(Boolean).join(" ")
}

// Not read yet is a guest until it is: a guest's links are words.
function guestOf(nameplate: Nameplate | null | undefined): boolean {
    return nameplate === null ? false : nameplate?.guest ?? true
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
    return translator().country(code)
}
