import {fakeAccountId} from "./fakeAccounts.ts"
import {NEWS_FROM_TILES} from "../domain/fortify.ts"
import {TIKTOK_PROFILE} from "../links.ts"
import {
    ChatAnnouncement,
    ChatBlockedError,
    ChatFlagRooms,
    ChatHistory,
    ChatHistoryGetter,
    ChatListener,
    ChatMessage,
    ChatMessageGoneError,
    ChatMutedError,
    ChatNotInRoomError,
    ChatRateLimitedError,
    ChatReactor,
    ChatRejectedError,
    ChatSeenMarker,
    ChatSender,
    countRunes,
    MAX_TEXT_LENGTH,
    MuteKind,
    OutgoingMessage,
    OutgoingReaction,
    Reaction,
    ReactionCount,
    ReactionsChange,
} from "./chat.ts";
import {NameColor} from "./player.ts";
import {ClosedRound} from "./standings.ts";
import {FlagMission} from "./missions.ts";
import {championOf} from "../domain/champion.ts";
import {podiumOf} from "../domain/roundReveal.ts";
import {quoteOf, withRemoved} from "../domain/chatLog.ts";
import {v4 as UUIDv4} from 'uuid';

const MESSAGES_PER_SECOND = 0.33
const MESSAGE_BURST = 5

export type FakeChatBackendOptions = {
    blocked?: boolean
    chatterIntervalMs?: number
    seenUntil?: number
}

const SEEN_AGO_MS = 150_000

// Their badges are the fake presence's, as the server's are player's.
type Chatter = {name: string, country: string, admin: boolean, color: NameColor, text: string, answers?: number}

const CHATTERS: Chatter[] = [
    {name: "Ana", country: "fr", admin: true, color: NameColor.PINK, text: "who keeps taking Brittany"},
    {name: "guest_91aa3d", country: "de", admin: false, color: NameColor.UNSPECIFIED, text: "we hold the north 💪", answers: 0},
    {name: "kiran_07", country: "in", admin: false, color: NameColor.UNSPECIFIED, text: `gm everyone, clips on ${TIKTOK_PROFILE}`},
    {name: "guest_aa1290", country: "jp", admin: false, color: NameColor.UNSPECIFIED, text: "the pacific is ours"},
]

const flagChatters = (flag: string): Chatter[] => [
    {name: "Ana", country: flag, admin: true, color: NameColor.PINK, text: "ok team, we push from the east"},
    {name: "guest_5be210", country: flag, admin: false, color: NameColor.UNSPECIFIED, text: "who has a bomb? their coast is weak", answers: 0},
]

const FLAG_ROOM_SLOWER = 3

const REPLIES_EVERY = 3

export const OWN_GUEST_NAME = "guest_c0ffee"

const ME = "me"

const BOT_REACTIONS = [Reaction.LAUGH, Reaction.CLOWN, Reaction.SKULL, Reaction.FIRE, Reaction.EARTH]

type Listener = {
    message: (message: ChatMessage) => void
    reactions?: (change: ReactionsChange) => void
    announcement?: (announcement: ChatAnnouncement) => void
    removed?: (messageId: string) => void
    left?: () => void
}

export class FakeChatBackend implements ChatSender, ChatHistoryGetter, ChatListener, ChatReactor, ChatSeenMarker, ChatFlagRooms {
    private readonly messages: ChatMessage[] = []
    private readonly announcements: ChatAnnouncement[] = []
    private readonly reactions = new Map<string, Map<Reaction, Set<string>>>()
    private readonly versions = new Map<string, number>()
    private readonly listeners = new Map<string, Listener>()
    private readonly timers: ReturnType<typeof setInterval>[] = []
    private readonly rooms = new Map<string, FakeChatBackend>()
    private readonly blocked: boolean
    private seenUntil: number
    private mutedUntil = 0
    private gifsMutedUntil = 0
    private tokens = MESSAGE_BURST
    private lastRefillMs = Date.now()
    private nextChatter = 0
    private left = false

    constructor(private readonly options: FakeChatBackendOptions = {}, private readonly world?: FakeChatBackend, flag?: string) {
        this.blocked = options.blocked ?? false
        this.seenUntil = options.seenUntil ?? Date.now() - SEEN_AGO_MS
        const chatters = flag === undefined ? CHATTERS : flagChatters(flag)

        chatters.forEach((chatter, index) => {
            this.publish({
                id: UUIDv4(),
                sentAt: Date.now() - (chatters.length - index) * 60_000,
                authorAccountId: fakeAccountId(chatter.name),
                authorName: chatter.name,
                authorAdmin: chatter.admin,
                authorColor: chatter.color,
                countryCode: chatter.country,
                text: chatter.text,
                reactions: [],
                reactionsVersion: 0,
                replyTo: chatter.answers === undefined ? undefined : quoteOf(this.messages[chatter.answers]),
            })
        })

        this.timers.push(setInterval(() => {
            const chatter = chatters[this.nextChatter % chatters.length]
            this.nextChatter++
            const answered = this.nextChatter % REPLIES_EVERY === 0
                ? this.messages[Math.floor(Math.random() * this.messages.length)]
                : undefined
            this.publish({
                id: UUIDv4(),
                sentAt: Date.now(),
                authorAccountId: fakeAccountId(chatter.name),
                authorName: chatter.name,
                authorAdmin: chatter.admin,
                authorColor: chatter.color,
                countryCode: chatter.country,
                text: `${chatter.text} (${this.nextChatter})`,
                reactions: [],
                reactionsVersion: 0,
                replyTo: answered && quoteOf(answered),
            })

            const target = this.messages[Math.floor(Math.random() * this.messages.length)]
            const reaction = BOT_REACTIONS[this.nextChatter % BOT_REACTIONS.length]
            this.give(target.id, reaction, chatter.name, true)
        }, (options.chatterIntervalMs ?? 8000) * (flag === undefined ? 1 : FLAG_ROOM_SLOWER)))
    }

    public flagRoom(flag: string): FakeChatBackend {
        const world = this.world ?? this
        const known = world.rooms.get(flag)
        if (known) return known

        const room = new FakeChatBackend(world.options, world, flag)
        world.rooms.set(flag, room)
        return room
    }

    // As when the player plays for another flag on another device: every flag room refuses it from now on.
    public leave(): string {
        const world = this.world ?? this
        world.rooms.forEach(room => {
            room.left = true
            room.listeners.forEach(listener => listener.left?.())
            room.listeners.clear()
        })
        return `left ${world.rooms.size} flag room(s)`
    }

    public close() {
        this.timers.forEach(clearInterval)
        this.timers.length = 0
        this.listeners.clear()
        this.rooms.forEach(room => room.close())
    }

    private refuse() {
        const shared = this.world ?? this
        if (this.left) throw new ChatNotInRoomError()
        if (this.blocked) throw new ChatBlockedError()
        if (Date.now() < shared.mutedUntil) throw new ChatMutedError(shared.mutedUntil)
    }

    public async sendMessage(message: OutgoingMessage): Promise<ChatMessage> {
        this.refuse()
        const gifsMutedUntil = (this.world ?? this).gifsMutedUntil
        if (message.gif && Date.now() < gifsMutedUntil) throw new ChatMutedError(gifsMutedUntil, 'gifs')

        const text = message.text.trim()
        if (message.gif ? text !== "" : text === "" || countRunes(text) > MAX_TEXT_LENGTH) throw new ChatRejectedError()

        const answered = this.messages.find(sent => sent.id === message.replyTo)
        if (message.replyTo !== undefined && !answered) throw new ChatMessageGoneError()

        if (!(this.world ?? this).allow()) throw new ChatRateLimitedError()

        const sent: ChatMessage = {
            id: UUIDv4(),
            sentAt: Date.now(),
            authorAccountId: fakeAccountId(OWN_GUEST_NAME),
            authorName: OWN_GUEST_NAME,
            authorAdmin: false,
            authorColor: NameColor.UNSPECIFIED,
            countryCode: message.countryCode,
            text,
            reactions: [],
            reactionsVersion: 0,
            replyTo: answered && quoteOf(answered),
            gif: message.gif,
        }

        this.publish(sent)
        return sent
    }

    public async react(reaction: OutgoingReaction): Promise<ReactionsChange> {
        this.refuse()
        if (!this.messages.some(message => message.id === reaction.messageId)) throw new ChatMessageGoneError()

        this.give(reaction.messageId, reaction.reaction, ME, reaction.on)
        return this.changeOf(reaction.messageId, ME)
    }

    public async getHistory(signal?: AbortSignal): Promise<ChatHistory> {
        signal?.throwIfAborted()
        if (this.left) throw new ChatNotInRoomError()
        return {
            messages: this.messages.map(message => ({
                ...message,
                reactions: this.tally(message.id, ME),
                reactionsVersion: this.versions.get(message.id) ?? 0,
            })),
            announcements: [...this.announcements],
            seenUntil: this.seenUntil,
        }
    }

    public async markSeen(until: number): Promise<void> {
        if (this.left) throw new ChatNotInRoomError()
        this.seenUntil = Math.max(this.seenUntil, Math.min(until, Date.now()))
    }

    public announceBomb(drop: {countryId: string, tile: number | undefined, cleared: readonly number[]}) {
        const announcement: ChatAnnouncement = {
            kind: "bomb",
            id: UUIDv4(),
            announcedAt: Date.now(),
            country: drop.countryId,
            tile: drop.tile,
            cleared: drop.cleared.length,
        }
        this.announce(announcement)
    }

    // As the server: only a sizeable territory is news in the chat.
    public announceFortify(fortified: {countryId: string, ground: string, landmass: number, tiles: number}) {
        if (fortified.tiles < NEWS_FROM_TILES) return
        this.announce({
            kind: "fortify",
            id: UUIDv4(),
            announcedAt: Date.now(),
            country: fortified.countryId,
            ground: fortified.ground,
            landmass: fortified.landmass,
            tiles: fortified.tiles,
        })
    }

    public announceRound(closed: ClosedRound) {
        this.announce({
            kind: "round",
            id: UUIDv4(),
            announcedAt: closed.endedAt,
            number: closed.number,
            finale: closed.finale,
            own: closed.moment !== "day",
            times: closed.times,
            podium: podiumOf(closed).map((standing) => ({
                country: standing.countryCode,
                rank: standing.rank,
                points: standing.points,
            })),
        })
    }

    // As the server: one line a season, however often it is told.
    public announceChampion(closed: ClosedRound) {
        const champion = championOf(closed)
        const id = `champion-${closed.season}`
        if (!champion || this.announcements.some((announcement) => announcement.id === id)) return
        this.announce({kind: "champion", id, announcedAt: closed.endedAt, season: champion.season, countries: champion.countryCodes})
    }

    // As the server: in the World and in the flag's own room, once.
    public announceFlagMission(flag: string, mission: FlagMission) {
        const world = this.world ?? this
        const id = `flag-mission-${flag}-${mission.id}`
        if (world.announcements.some((announcement) => announcement.id === id)) return
        const announcement: ChatAnnouncement = {
            kind: "flag_mission", id, announcedAt: Date.now(), country: flag, points: mission.points, order: mission.order,
        }
        world.announce(announcement)
        world.flagRoom(flag).announce(announcement)
    }

    private announce(announcement: ChatAnnouncement) {
        this.announcements.push(announcement)
        this.listeners.forEach(listener => listener.announcement?.(announcement))
    }

    public mute(seconds = 3600, name = OWN_GUEST_NAME, kind: MuteKind = 'chat') {
        const until = Date.now() + seconds * 1000
        const world = this.world ?? this
        if (name === OWN_GUEST_NAME && kind === 'gifs') world.gifsMutedUntil = until
        if (name === OWN_GUEST_NAME && kind === 'chat') world.mutedUntil = until
        this.announce({kind: "mute", id: UUIDv4(), announcedAt: Date.now(), name, muteKind: kind, seconds})
    }

    public remove(messageId = this.messages[this.messages.length - 1]?.id): string {
        if (!this.messages.some(message => message.id === messageId)) return "no such message"

        this.messages.splice(0, this.messages.length, ...withRemoved(this.messages, messageId))
        this.listeners.forEach(listener => listener.removed?.(messageId))
        return `removed ${messageId}`
    }

    public listenForMessages(
        callback: (message: ChatMessage) => void,
        onReactions?: (change: ReactionsChange) => void,
        onAnnouncement?: (announcement: ChatAnnouncement) => void,
        _onResumed?: () => void,
        onRemoved?: (messageId: string) => void,
        onLeft?: () => void,
    ): () => void {
        if (this.left) {
            onLeft?.()
            return () => {}
        }
        const id = UUIDv4()
        this.listeners.set(id, {
            message: callback, reactions: onReactions, announcement: onAnnouncement, removed: onRemoved, left: onLeft,
        })
        return () => {
            this.listeners.delete(id)
        }
    }

    private publish(message: ChatMessage) {
        this.messages.push(message)
        this.listeners.forEach(listener => listener.message(message))
    }

    private give(messageId: string, reaction: Reaction, reactor: string, on: boolean) {
        const given = this.reactions.get(messageId) ?? new Map<Reaction, Set<string>>()
        this.reactions.set(messageId, given)
        const reactors = given.get(reaction) ?? new Set<string>()
        if (reactors.has(reactor) === on) return

        if (on) reactors.add(reactor)
        else reactors.delete(reactor)
        if (reactors.size === 0) given.delete(reaction)
        else given.set(reaction, reactors)

        this.versions.set(messageId, (this.versions.get(messageId) ?? 0) + 1)
        const change = this.changeOf(messageId, undefined)
        this.listeners.forEach(listener => listener.reactions?.(change))
    }

    private changeOf(messageId: string, viewer: string | undefined): ReactionsChange {
        return {messageId, reactions: this.tally(messageId, viewer), version: this.versions.get(messageId) ?? 0}
    }

    private tally(messageId: string, viewer: string | undefined): ReactionCount[] {
        return [...(this.reactions.get(messageId) ?? new Map<Reaction, Set<string>>())]
            .map(([reaction, reactors]) => ({
                reaction,
                count: reactors.size,
                mine: viewer !== undefined && reactors.has(viewer),
                reactors: [...reactors].map(named),
            }))
    }

    private allow(): boolean {
        const now = Date.now()
        this.tokens = Math.min(
            MESSAGE_BURST,
            this.tokens + ((now - this.lastRefillMs) / 1000) * MESSAGES_PER_SECOND,
        )
        this.lastRefillMs = now

        if (this.tokens < 1) return false
        this.tokens -= 1
        return true
    }
}

function named(reactor: string): string {
    return reactor === ME ? OWN_GUEST_NAME : reactor
}
